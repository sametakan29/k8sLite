package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"sync"
	"time"
)

// Nomad Veri Modelleri
type Task struct {
	Name    string   `json:"name"`
	Driver  string   `json:"driver"`  // "raw_exec" veya "docker"
	Command string   `json:"command"` // Çalıştırılacak komut (örn: "ping", "powershell")
	Args    []string `json:"args"`    // Parametreler
}

type Job struct {
	ID     string `json:"id"`
	NodeID string `json:"node_id,omitempty"`
	Status string `json:"status"` // "pending", "running", "completed"
	Task   Task   `json:"task"`
}

type ClientNode struct {
	ID      string `json:"id"`
	Address string `json:"address"`
}

// -----------------------------------------------------------------------------
// SERVER MODU (Nomad Server / Scheduler / Store)
// -----------------------------------------------------------------------------
type ServerState struct {
	mu      sync.Mutex
	jobs    map[string]*Job
	clients map[string]ClientNode
	rrIndex int
}

func runServer(port string) {
	state := &ServerState{
		jobs:    make(map[string]*Job),
		clients: make(map[string]ClientNode),
	}

	// 1. Client Node Register Endpoint'i
	http.HandleFunc("/v1/client/register", func(w http.ResponseWriter, r *http.Request) {
		var c ClientNode
		if err := json.NewDecoder(r.Body).Decode(&c); err != nil {
			http.Error(w, err.Error(), 400)
			return
		}
		state.mu.Lock()
		state.clients[c.ID] = c
		state.mu.Unlock()
		log.Printf("[Server] Yeni client kaydedildi: %s (%s)\n", c.ID, c.Address)
		w.WriteHeader(http.StatusOK)
	})

	// 2. Job Submit & List Endpoint'i
	http.HandleFunc("/v1/jobs", func(w http.ResponseWriter, r *http.Request) {
		state.mu.Lock()
		defer state.mu.Unlock()

		if r.Method == http.MethodPost {
			var job Job
			if err := json.NewDecoder(r.Body).Decode(&job); err != nil {
				http.Error(w, err.Error(), 400)
				return
			}
			job.Status = "pending"
			state.jobs[job.ID] = &job
			log.Printf("[Server] Job kabul edildi: '%s' (Driver: %s)\n", job.ID, job.Task.Driver)
			w.WriteHeader(http.StatusCreated)
			return
		}

		// GET
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(state.jobs)
	})

	// 3. Client'ların kendi işlerini çekmesi (Heartbeat / Sync)
	http.HandleFunc("/v1/client/jobs", func(w http.ResponseWriter, r *http.Request) {
		nodeID := r.URL.Query().Get("node")
		state.mu.Lock()
		defer state.mu.Unlock()

		assigned := []*Job{}
		for _, j := range state.jobs {
			if j.NodeID == nodeID {
				assigned = append(assigned, j)
			}
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(assigned)
	})

	// Gömülü Scheduler Döngüsü (Nomad usulü tek binary içinde!)
	go func() {
		for {
			time.Sleep(2 * time.Second)
			state.mu.Lock()
			clientList := make([]ClientNode, 0, len(state.clients))
			for _, c := range state.clients {
				clientList = append(clientList, c)
			}

			if len(clientList) > 0 {
				for _, job := range state.jobs {
					if job.NodeID == "" && job.Status == "pending" {
						// Round-robin yerleştir
						target := clientList[state.rrIndex%len(clientList)]
						state.rrIndex++
						job.NodeID = target.ID
						log.Printf("[Scheduler] Job '%s' -> Node '%s' üzerine atandı!\n", job.ID, target.ID)
					}
				}
			}
			state.mu.Unlock()
		}
	}()

	log.Printf("[Nomad-Lite Server] http://localhost:%s adresinde dinliyor...\n", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}

// -----------------------------------------------------------------------------
// CLIENT MODU (Nomad Client / Raw Exec Driver)
// -----------------------------------------------------------------------------
func runClient(nodeID, serverAddr string) {
	log.Printf("[Nomad-Lite Client] %s baslatildi. Server'a baglaniliyor...\n", nodeID)

	// Server'a kendini tanıt
	regBody, _ := json.Marshal(ClientNode{ID: nodeID, Address: "localhost"})
	_, err := http.Post(fmt.Sprintf("%s/v1/client/register", serverAddr), "application/json", bytes.NewBuffer(regBody))
	if err != nil {
		log.Fatalf("Server'a baglanilamadi: %v", err)
	}

	executedJobs := make(map[string]bool)

	for {
		time.Sleep(2 * time.Second)
		resp, err := http.Get(fmt.Sprintf("%s/v1/client/jobs?node=%s", serverAddr, nodeID))
		if err != nil {
			continue
		}

		var jobs []*Job
		json.NewDecoder(resp.Body).Decode(&jobs)
		resp.Body.Close()

		for _, job := range jobs {
			if executedJobs[job.ID] {
				continue
			}
			executedJobs[job.ID] = true

			log.Printf("[Client-%s] Yeni is alindi: '%s'. Driver: %s\n", nodeID, job.ID, job.Task.Driver)

			// Nomad'in Alametifarikası: RAW EXEC DRIVER!
			go func(j *Job) {
				if j.Task.Driver == "raw_exec" {
					log.Printf("[Client-%s] '%s %v' isletim sisteminde kosturuluyor...\n", nodeID, j.Task.Command, j.Task.Args)
					cmd := exec.Command(j.Task.Command, j.Task.Args...)
					cmd.Stdout = os.Stdout
					cmd.Stderr = os.Stderr
					if err := cmd.Run(); err != nil {
						log.Printf("[Client-%s] Job '%s' hata verdi: %v\n", nodeID, j.ID, err)
					} else {
						log.Printf("[Client-%s] Job '%s' basariyla tamamlandi (Exit 0)!\n", nodeID, j.ID)
					}
				}
			}(job)
		}
	}
}

// -----------------------------------------------------------------------------
// CLI GİRİŞ NOKTASI
// -----------------------------------------------------------------------------
func main() {
	serverFlag := flag.Bool("server", false, "Server modunda calistir")
	clientFlag := flag.Bool("client", false, "Client modunda calistir")
	nodeName := flag.String("node", "node-alpha", "Client Node ID")
	serverAddr := flag.String("server-addr", "http://localhost:4646", "Server adresi")
	port := flag.String("port", "4646", "Server portu (Default 4646 - Nomad standardı)")

	flag.Parse()

	if *serverFlag {
		runServer(*port)
	} else if *clientFlag {
		runClient(*nodeName, *serverAddr)
	} else {
		// Basit CLI: argüman olarak "status" veya "run" verilebilir
		args := flag.Args()
		if len(args) == 0 {
			fmt.Println("Kullanim:")
			fmt.Println("  go run ./cmd/nomad-lite -server")
			fmt.Println("  go run ./cmd/nomad-lite -client -node=client-1")
			fmt.Println("  go run ./cmd/nomad-lite status")
			fmt.Println("  go run ./cmd/nomad-lite run <job.json>")
			return
		}

		switch args[0] {
		case "status":
			resp, err := http.Get("http://localhost:4646/v1/jobs")
			if err != nil {
				log.Fatalf("Server kapali: %v", err)
			}
			body, _ := io.ReadAll(resp.Body)
			fmt.Println(string(body))

		case "run":
			if len(args) < 2 {
				log.Fatal("Lutfen bir job dosyasi belirtin (orn: job.json)")
			}
			data, err := os.ReadFile(args[1])
			if err != nil {
				log.Fatalf("Dosya okunamadi: %v", err)
			}
			resp, err := http.Post("http://localhost:4646/v1/jobs", "application/json", bytes.NewBuffer(data))
			if err != nil {
				log.Fatalf("Job gonderilemedi: %v", err)
			}
			if resp.StatusCode == 201 {
				fmt.Println("Job basariyla gonderildi (Accepted)!")
			}
		}
	}
}
