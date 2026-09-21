package main

import (
	"flag"
	"fmt"
	"log"
	"time"

	"k8s-lite/pkg/api"
)

func main() {
	// Terminalden node adı ve adresi alabilmek için CLI flag'leri tanımlıyoruz
	nodeName := flag.String("node", "node-1", "Calisacak node adi")
	nodeAddr := flag.String("addr", "localhost:10250", "Node adresi")
	flag.Parse()

	client := api.NewClient("http://localhost:8080")

	// 1. Önce bu node'u kümeye kaydet
	node := &api.Node{
		Name:    *nodeName,
		Address: *nodeAddr,
	}
	if err := client.RegisterNode(node); err != nil {
		log.Fatalf("[Kubelet-%s] Node kaydedilemedi: %v\n", *nodeName, err)
	}
	fmt.Printf("[Kubelet-%s] Basariyla kaydedildi. Gorevler dinleniyor...\n", *nodeName)

	// 2. Reconciliation döngüsü: Her 2 saniyede bir kendi pod'larını kontrol et
	for {
		time.Sleep(2 * time.Second)

		pods, err := client.ListPods()
		if err != nil {
			log.Printf("[Kubelet-%s] Podlar alinamadi: %v\n", *nodeName, err)
			continue
		}

		for _, pod := range pods {
			// Sadece bu node'a ait pod'larla ilgilen
			if pod.NodeName != *nodeName {
				continue
			}

			// Durum 1: Pod silinme sürecinde mi? (Graceful shutdown simülasyonu)
			if pod.DeletionTimestamp != nil || pod.Phase == api.PodTerminating {
				fmt.Printf("[Kubelet-%s] Pod '%s' siliniyor, container durduruldu.\n", *nodeName, pod.Name)
				pod.Phase = api.PodDeleted
				if err := client.UpdatePod(pod); err != nil {
					log.Printf("[Kubelet-%s] Pod silme guncellemesi basarisiz: %v\n", *nodeName, err)
				}
				continue
			}

			// Durum 2: Scheduler buraya yeni bir pod atamış mı?
			if pod.Phase == api.PodScheduled {
				fmt.Printf("[Kubelet-%s] Yeni pod yakalandi: '%s' (Image: %s). Container baslatiliyor...\n", *nodeName, pod.Name, pod.Image)
				time.Sleep(1 * time.Second) // Başlatma simülasyonu
				pod.Phase = api.PodRunning

				if err := client.UpdatePod(pod); err != nil {
					log.Printf("[Kubelet-%s] Pod durumu guncellenemedi: %v\n", *nodeName, err)
				} else {
					fmt.Printf("[Kubelet-%s] Pod '%s' artik RUNNING modunda!\n", *nodeName, pod.Name)
				}
			}
		}
	}
}
