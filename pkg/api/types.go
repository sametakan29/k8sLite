package api

import "time"

//pod'un yaşam döngüsü aşamaları
type PodPhase string

const (
	PodPending     PodPhase = "Pending"     // Oluşturuldu, henüz bir node atanmadı
	PodScheduled   PodPhase = "Scheduled"   // Scheduler bir node atadı
	PodRunning     PodPhase = "Running"     // Kubelet pod'u çalıştırdı
	PodTerminating PodPhase = "Terminating" // Silme istendi, temizlik bekleniyor
	PodDeleted     PodPhase = "Deleted"     // Tamamen kaldırıldı
)

// Pod tanımı
type Pod struct {
	Name              string     `json:"name"`
	Namespace         string     `json:"namespace"`
	Image             string     `json:"image"`
	NodeName          string     `json:"nodeName,omitempty"` // Hangi node'a atandığı
	Phase             PodPhase   `json:"phase"`
	DeletionTimestamp *time.Time `json:"deletionTimestamp,omitempty"` // Graceful silme için
}

// Node tanımı
type Node struct {
	Name    string `json:"name"`
	Address string `json:"address"`
	Status  string `json:"status"` // "Ready", vs.
}
