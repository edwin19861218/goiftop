package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"github.com/edwin19861218/goiftop/db"
	"github.com/edwin19861218/goiftop/utils/log"
	"github.com/google/gopacket"
	"github.com/google/gopacket/pcap"
	"math"
	"net/http"
	"os"
	"os/signal"
	"sort"
	"strconv"
	"strings"
	"syscall"
	"time"
)

var ifaceName string
var filter string
var enableLayer4 bool
var port int
var isShowVersion bool
var mode string
var isSilence bool
var serverUri string
var token string
var dbUri string
var dbClient *db.InfluxDB
var cache *FlowSnapshotCache

const (
	modeServer = "server"
	modeSingle = "single"
	modeClient = "client"
)

func init() {
	flag.StringVar(&ifaceName, "i", "", "Interface name")
	flag.StringVar(&filter, "bpf", "", "BPF filter")
	flag.BoolVar(&enableLayer4, "l4", false, "Show transport layer flows")
	flag.IntVar(&port, "p", 16384, "Http server listening port")
	flag.BoolVar(&isShowVersion, "v", false, "Version")
	flag.StringVar(&mode, "m", modeSingle, "Running in server mode, client mode or just as a single node")
	flag.BoolVar(&isSilence, "s", false, "Print output")
	flag.StringVar(&serverUri, "uri", "", "Server uri")
	flag.StringVar(&dbUri, "db", "", "DB uri")
	flag.StringVar(&token, "token", "", "Token to validate store api")

	flag.Parse()

	if isShowVersion {
		fmt.Println(AppVersion)
		closeApp("", 0)
	}

	if ifaceName == "" {
		//DO NOT use net.Interfaces() for compatibility error in windows
		if ifaces, err := pcap.FindAllDevs(); err == nil {
			for _, iface := range ifaces {
				if strings.HasPrefix(iface.Name, "zt") {
					log.Infof("auto bind linux interface %s %s with ip %v", iface.Name, iface.Description, iface.Addresses)
					ifaceName = iface.Name
					break
				}
				if strings.HasPrefix(iface.Description, "ZeroTier") && !strings.Contains(iface.Description, "Packet") && !strings.Contains(iface.Description, "Filter") {
					log.Infof("auto windows interface %s %s with ip %v", iface.Name, iface.Description, iface.Addresses)
					ifaceName = iface.Name
					break
				}
			}
		}
	}
	cache = NewCache()
	if mode == modeServer {
		client, err := db.New(dbUri)
		if err != nil {
			log.Error("error server db config", err)
			closeApp("error config", 1)
		}
		dbClient = client
		if serverUri == "" {
			serverUri = fmt.Sprintf("http://127.0.0.1:%d/store", port)
		}
	} else if mode == modeClient {
		if serverUri == "" {
			log.Error("error server config")
			closeApp("error config", 1)
		}
	}
	log.Infof("running in %s mode", mode)
}

func main() {
	go func() {
		log.Infof("Start %s mode server on port %d with interface %s", mode, port, ifaceName)
		http.HandleFunc("/l3flow", L3FlowHandler)
		http.HandleFunc("/l4flow", L4FlowHandler)
		http.HandleFunc("/store", StoreHandler)

		http.Handle("/", http.StripPrefix("/", http.FileServer(assetFS())))

		err := http.ListenAndServe(":"+strconv.Itoa(port), nil)
		if err != nil {
			log.Errorf("Failed to start http server with error: %s" + err.Error())
			closeApp("Failed to start http server", 1)
		}
	}()

	if os.Geteuid() != 0 {
		log.Errorln("Must run as root")
	}

	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, syscall.SIGHUP, syscall.SIGINT)
	tickStatsDuration := time.Tick(time.Duration(1) * time.Second)
	clientStatsDuration := time.Tick(time.Duration(10) * time.Second)

	Stats.ifaces[ifaceName] = NewIface(ifaceName)
	ctx, cancel := context.WithCancel(context.Background())
	go listenPacket(ifaceName, ctx)

	for {
		select {
		case <-tickStatsDuration:
			updateL3FlowSnapshots(cache)
			if enableLayer4 {
				updateL4FlowSnapshots(cache)
			}
			if !isSilence {
				fmt.Println("------")
				printFlowSnapshots(cache.L3FlowSnapshots)
				if enableLayer4 {
					fmt.Println()
					printFlowSnapshots(cache.L4FlowSnapshots)
				}
			}
		case <-clientStatsDuration:
			if mode != modeSingle {
				//send to influx
				var data []*FlowSnapshot
				if enableLayer4 {
					data = cache.L4FlowSnapshots
				} else {
					data = cache.L3FlowSnapshots
				}
				if len(data) == 0 {
					continue
				}
				resJson, err := json.Marshal(data)
				if err != nil {
					log.Error("json format error", err)
					continue
				}
				body := bytes.NewBuffer(resJson)
				resp, err := http.Post(fmt.Sprintf("%s?token=%s", serverUri, token), "application/json;charset=utf-8", body)
				if err != nil {
					log.Error("Post failed", err)
					continue
				}
				defer resp.Body.Close()
			}
		case <-signalChan:
			cancel()
			closeApp("", 0)
		}
	}

}

func closeApp(msg string, code int) {
	log.Infof("App Exit with msg '%s'", msg)
	if dbClient != nil {
		dbClient.Close()
	}
	os.Exit(code)
}

func listenPacket(ifaceName string, ctx context.Context) {
	handle, err := pcap.OpenLive(ifaceName, 65536, true, pcap.BlockForever)
	if err != nil {
		log.Errorf("Failed to OpenLive by pcap, err: %s\n", err.Error())
		closeApp("pcap error", 1)
	}

	err = handle.SetBPFFilter(filter)
	if err != nil {
		log.Errorf("Failed to set BPF filter, err: %s\n", err.Error())
		closeApp("bpf filter error", 1)
	}

	defer handle.Close()

	ps := gopacket.NewPacketSource(handle, handle.LinkType())
	for {
		select {
		case <-ctx.Done():
			return
		case p := <-ps.Packets():
			go Stats.PacketHandler(ifaceName, p)
		}
	}
}

func updateL3FlowSnapshots(cache *FlowSnapshotCache) {
	L3FlowSnapshots := make([]*FlowSnapshot, 0, 0)
	Stats.ifaces[ifaceName].UpdateL3FlowQueue()
	Stats.ifaces[ifaceName].Lock.Lock()
	for _, v := range Stats.ifaces[ifaceName].L3Flows {
		fss := v.GetSnapshot()
		if fss.DownStreamRate1+fss.UpStreamRate1+fss.DownStreamRate15+fss.UpStreamRate15+fss.DownStreamRate60+fss.UpStreamRate60 > 0 {
			L3FlowSnapshots = append(L3FlowSnapshots, fss)
		}
	}
	Stats.ifaces[ifaceName].Lock.Unlock()
	sort.Slice(L3FlowSnapshots, func(i, j int) bool {
		return math.Max(float64(L3FlowSnapshots[i].UpStreamRate1), float64(L3FlowSnapshots[i].DownStreamRate1)) >
			math.Max(float64(L3FlowSnapshots[j].UpStreamRate1), float64(L3FlowSnapshots[j].DownStreamRate1))
	})
	cache.L3FlowSnapshots = L3FlowSnapshots
}

func updateL4FlowSnapshots(cache *FlowSnapshotCache) {
	L4FlowSnapshots := make([]*FlowSnapshot, 0, 0)
	Stats.ifaces[ifaceName].UpdateL4FlowQueue()
	Stats.ifaces[ifaceName].Lock.Lock()
	for _, v := range Stats.ifaces[ifaceName].L4Flows {
		fss := v.GetSnapshot()
		if fss.DownStreamRate1+fss.UpStreamRate1+fss.DownStreamRate15+fss.UpStreamRate15+fss.DownStreamRate60+fss.UpStreamRate60 > 0 {
			L4FlowSnapshots = append(L4FlowSnapshots, fss)
		}
	}
	Stats.ifaces[ifaceName].Lock.Unlock()
	sort.Slice(L4FlowSnapshots, func(i, j int) bool {
		return math.Max(float64(L4FlowSnapshots[i].UpStreamRate1), float64(L4FlowSnapshots[i].DownStreamRate1)) >
			math.Max(float64(L4FlowSnapshots[j].UpStreamRate1), float64(L4FlowSnapshots[j].DownStreamRate1))
	})
	cache.L4FlowSnapshots = L4FlowSnapshots
}

func printFlowSnapshots(flowSnapshots []*FlowSnapshot) {
	if len(flowSnapshots) > 0 {
		fmt.Printf("%-8s %-32s %-32s %-16s %-16s %-16s %-16s %-16s %-16s\n", "Protocol", "Src", "Dst", "Up1", "Down1", "Up15", "Down15", "Up60", "Down60")
	}

	for _, f := range flowSnapshots {
		u1 := rateToStr(f.UpStreamRate1)
		d1 := rateToStr(f.DownStreamRate1)
		u15 := rateToStr(f.UpStreamRate15)
		d15 := rateToStr(f.DownStreamRate15)
		u60 := rateToStr(f.UpStreamRate60)
		d60 := rateToStr(f.DownStreamRate60)
		fmt.Printf("%-8s %-32s %-32s %-16s %-16s %-16s %-16s %-16s %-16s\n", f.Protocol, f.SourceAddress, f.DestinationAddress, u1, d1, u15, d15, u60, d60)
	}
}

func rateToStr(rate int64) (rs string) {
	if rate >= 1000000 {
		rs = fmt.Sprintf("%.2f Mbps", float64(rate)/float64(1000000))
	} else if rate >= 1000 && rate < 1000000 {
		rs = fmt.Sprintf("%.2f Kbps", float64(rate)/float64(1000))
	} else {
		rs = fmt.Sprintf("%d bps", rate)
	}

	return
}
