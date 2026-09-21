package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"k8s-lite/pkg/api"
)

func main() {
	client := api.NewClient("http://localhost:8080")
	manifestDir := "./manifests"

	fmt.Println("[ArgoCD-Lite] GitOps Controller baslatildi.")
	fmt.Printf("[ArgoCD-Lite] '%s' klasoru (Single Source of Truth) izleniyor...\n", manifestDir)

	for {
		time.Sleep(3 * time.Second)

		// 1. Desired State: Klasördeki dosyaları oku
		desiredPods, err := loadManifests(manifestDir)
		if err != nil {
			log.Printf("[ArgoCD-Lite] Manifest okuma hatasi: %v\n", err)
			continue
		}

		// 2. Actual State: Kümeye sor, şu an ne var?
		actualPodsList, err := client.ListPods()
		if err != nil {
			log.Printf("[ArgoCD-Lite] Kume durumu alinamadi: %v\n", err)
			continue
		}

		// Hızlı arama için map'e çevirelim
		actualPods := make(map[string]*api.Pod)
		for _, p := range actualPodsList {
			actualPods[p.Name] = p
		}

		// ----------------------------------------------------
		// AŞAMA 1: Auto-Sync & Self-Healing
		// Dosyalarda var ama kümede yoksa (veya bozulmuşsa) düzelt!
		// ----------------------------------------------------
		for name, desired := range desiredPods {
			actual, exists := actualPods[name]

			if !exists {
				// Drift tespit edildi: Git'te var ama kümede yok!
				fmt.Printf("[ArgoCD-Lite] [OutOfSync] Pod '%s' karsilastirildi: Kumede EKSIK! Olusturuluyor...\n", name)
				err := client.CreatePod(desired)
				if err != nil {
					log.Printf("[ArgoCD-Lite] Pod olusturma hatasi: %v\n", err)
				} else {
					fmt.Printf("[ArgoCD-Lite] [Synced] Pod '%s' basariyla kumeye senkronize edildi.\n", name)
				}
			} else if actual.Image != desired.Image {
				// Drift tespit edildi: Biri kümedeki imajı elle değiştirmiş!
				fmt.Printf("[ArgoCD-Lite] [OutOfSync] Pod '%s' imaj uyusmazligi! (Kume: %s, Git: %s). Self-Healing baslatiliyor...\n", name, actual.Image, desired.Image)
				actual.Image = desired.Image
				_ = client.UpdatePod(actual)
			}
		}

		// ----------------------------------------------------
		// AŞAMA 2: Pruning (Temizlik)
		// Kümede var ama dosyalardan silinmişse kümeden de kaldır!
		// ----------------------------------------------------
		for name, actual := range actualPods {
			if _, exists := desiredPods[name]; !exists {
				// Pod silinme sürecinde değilse silme komutu ver
				if actual.DeletionTimestamp == nil && actual.Phase != api.PodTerminating {
					fmt.Printf("[ArgoCD-Lite] [Prune] Pod '%s' manifest klasorunde artik YOK! Kumeden siliniyor...\n", name)
					err := client.DeletePod("default", name)
					if err != nil {
						log.Printf("[ArgoCD-Lite] Prune hatasi: %v\n", err)
					}
				}
			}
		}
	}
}

// manifests/ klasöründeki tüm .json dosyalarını okur
func loadManifests(dir string) (map[string]*api.Pod, error) {
	pods := make(map[string]*api.Pod)

	files, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	for _, f := range files {
		if filepath.Ext(f.Name()) == ".json" {
			data, err := os.ReadFile(filepath.Join(dir, f.Name()))
			if err != nil {
				continue
			}

			var pod api.Pod
			if err := json.Unmarshal(data, &pod); err == nil {
				pod.Namespace = "default"
				pods[pod.Name] = &pod
			}
		}
	}
	return pods, nil
}
