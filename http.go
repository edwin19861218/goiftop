package main

import (
	"encoding/json"
	"io/ioutil"

	"net/http"
)

func L3FlowHandler(w http.ResponseWriter, r *http.Request) {
	resJson, err := json.Marshal(cache.L3FlowSnapshots)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write(resJson)
}

func L4FlowHandler(w http.ResponseWriter, r *http.Request) {
	resJson, err := json.Marshal(cache.L4FlowSnapshots)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write(resJson)
}

func StoreHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "only support Post", 400)
		return
	}
	if mode == modeServer && dbClient == nil {
		http.Error(w, "only available in server mode", 400)
		return
	}
	if r.FormValue("token") != token {
		http.Error(w, "error token", 400)
		return
	}
	b, err := ioutil.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "error data", 400)
		return
	}
	defer r.Body.Close()
	var snapshot []FlowSnapshot
	err = json.Unmarshal(b, &snapshot)
	if err != nil {
		http.Error(w, "error data", 400)
		return
	}
	for _, ss := range snapshot {
		dbClient.Write(ss.Protocol, ss.SourceAddress, ss.DestinationAddress, ss.UpStreamRate60, ss.DownStreamRate60)
	}
	dbClient.WriteFlush()
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(200)
}
