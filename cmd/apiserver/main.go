package main

import (
	"encoding/json"
	"fmt"
	"k8s-lite/pkg/api"
	"k8s-lite/pkg/store"
	"log"
	"net/http"
	"strings"
)

var clusterStore = store.NewMemoryStore()

func main() {
	mux := http.NewServeMux()

	mux.HandleFunc("/pods", handlePods)
	mux.HandleFunc("/pods/", handlePodDetail)
	mux.HandleFunc("/nodes", handleNodes)

	port := ":8080"
	fmt.Printf("[APIServer] Dinleniyor: http://localhost%s\n", port)
	log.Fatal(http.ListenAndServe(port, mux))
}
func handlePods(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	switch r.Method {
	case http.MethodGet:
		pods := clusterStore.ListPods()
		json.NewEncoder(w).Encode(pods)

	case http.MethodPost:
		var pod api.Pod
		if err := json.NewDecoder(r.Body).Decode(&pod); err != nil {
			http.Error(w, "Geçersiz istek gövdesi", http.StatusBadRequest)
			return
		}
		if pod.Namespace == "" {
			pod.Namespace = "default"
		}
		if err := clusterStore.CreatePod(&pod); err != nil {
			http.Error(w, err.Error(), http.StatusConflict)
			return
		}
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(pod)

	default:
		http.Error(w, "Metot desteklenmiyor", http.StatusMethodNotAllowed)
	}
}
func handlePodDetail(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// URL parçalama: /pods/{namespace}/{name}
	parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(parts) < 3 {
		http.Error(w, "Geçersiz path formatı (/pods/{ns}/{name} olmalı)", http.StatusBadRequest)
		return
	}
	namespace := parts[1]
	name := parts[2]

	switch r.Method {
	case http.MethodPut:
		var pod api.Pod
		if err := json.NewDecoder(r.Body).Decode(&pod); err != nil {
			http.Error(w, "Geçersiz veri", http.StatusBadRequest)
			return
		}
		pod.Namespace = namespace
		pod.Name = name

		// Eğer pod zaten Deleted durumuna geldiyse hafızadan tamamen düşür (hard delete)
		if pod.Phase == api.PodDeleted {
			_ = clusterStore.HardDeletePod(namespace, name)
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]string{"status": "Deleted"})
			return
		}

		if err := clusterStore.UpdatePod(&pod); err != nil {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		json.NewEncoder(w).Encode(pod)

	case http.MethodDelete:
		if err := clusterStore.DeletePod(namespace, name); err != nil {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"status": "Terminating"})

	default:
		http.Error(w, "Metot desteklenmiyor", http.StatusMethodNotAllowed)
	}
}
func handleNodes(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	switch r.Method {
	case http.MethodGet:
		nodes := clusterStore.ListNodes()
		json.NewEncoder(w).Encode(nodes)

	case http.MethodPost:
		var node api.Node
		if err := json.NewDecoder(r.Body).Decode(&node); err != nil {
			http.Error(w, "Geçersiz veri", http.StatusBadRequest)
			return
		}
		clusterStore.RegisterNode(&node)
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(node)

	default:
		http.Error(w, "Metot desteklenmiyor", http.StatusMethodNotAllowed)
	}
}
