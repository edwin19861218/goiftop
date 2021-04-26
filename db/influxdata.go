package db

import (
	"fmt"
	"github.com/edwin19861218/goiftop/utils/log"
	"github.com/influxdata/influxdb-client-go/v2"
	"github.com/influxdata/influxdb-client-go/v2/api"
	"net/url"
)

type InfluxDB struct {
	url      string
	token    string
	bucket   string
	org      string
	client   influxdb2.Client
	writeAPI api.WriteAPI
}

//uri http://10.8.1.132:8086/?token=??&bucket=??&org=??
func New(uri string) (*InfluxDB, error) {
	u, err := url.Parse(uri)
	if err != nil {
		log.Fatal("error db uri", err)
		return nil, err
	}
	q := u.Query()
	db := &InfluxDB{url: fmt.Sprintf("%s://%s%s", u.Scheme, u.Host, u.Path), token: q.Get("token"), bucket: q.Get("bucket"), org: q.Get("org")}
	db.client = influxdb2.NewClient(db.url, db.token)
	db.writeAPI = db.client.WriteAPI(db.org, db.bucket)
	log.Infof("init influx server %s", u.Host)
	return db, err
}

func (db *InfluxDB) Write(protocol, from, to string, up, down int64) {
	msg := fmt.Sprintf("flow,protocol=%s,from=%s,to=%s up=%d,down=%d", protocol, from, to, up, down)
	log.Debugf("store msg %s", msg)
	db.writeAPI.WriteRecord(msg)
}

func (db *InfluxDB) WriteFlush() {
	db.writeAPI.Flush()
}

func (db *InfluxDB) Close() {
	if db.client != nil {
		db.client.Close()
	}
}
