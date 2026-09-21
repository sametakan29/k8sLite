package main

import (
	"fmt"
	"log"
	"time"

	"k8s-lite/pkg/api"
)

func main() {
	client := api.NewClient("http://localhost:8080")
	fmt.Println("[Scheduler] Baslatildi, sahipsiz podlar taranıyor...")

	nodeIndex := 0

	// Sonsuz döngü: Her 2 saniyede bir apiserver'ı kontrol et (Reconciliation)
	for {
		time.Sleep(2 * time.Second)

		// 1. Kümeye kayıtlı node'ları al
		nodes, err := client.ListNodes()
		if err != nil {
			log.Printf("[Scheduler] Node listesi alinamadi: %v\n", err)
			continue
		}

		if len(nodes) == 0 {
			// Henüz hazır bir node yoksa pod dağıtılamaz
			continue
		}

		// 2. Pod'ları kontrol et
		pods, err := client.ListPods()
		if err != nil {
			log.Printf("[Scheduler] Pod listesi alinamadi: %v\n", err)
			continue
		}

		for _, pod := range pods {
			// Sadece atanmamış ve silinme sürecinde olmayan pod'ları yakala
			if pod.Phase == api.PodPending && pod.NodeName == "" && pod.DeletionTimestamp == nil {
				// Round-robin ile bir node seç
				selectedNode := nodes[nodeIndex%len(nodes)]
				nodeIndex++

				// Pod'u güncelle
				pod.NodeName = selectedNode.Name
				pod.Phase = api.PodScheduled

				err := client.UpdatePod(pod)
				if err != nil {
					log.Printf("[Scheduler] Pod atamasi basarisiz (%s -> %s): %v\n", pod.Name, selectedNode.Name, err)
				} else {
					fmt.Printf("[Scheduler] Pod '%s' basariyla '%s' node'una atandi.\n", pod.Name, selectedNode.Name)
				}
			}
		}
	}
}
