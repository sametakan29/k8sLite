package store

import (
	"errors"
	"k8s-lite/pkg/api"
	"sync"
	"time"
)

type MemoryStore struct {
	mu    sync.RWMutex
	pods  map[string]*api.Pod  // key: "namespace/name" (Örn: "default/mypod")
	nodes map[string]*api.Node // key: "node-name"
}

// Yeni boş bir depo başlatıcı
func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		pods:  make(map[string]*api.Pod),
		nodes: make(map[string]*api.Node),
	}
}

// ------------------- POD İŞLEMLERİ -------------------

// Yeni pod ekleme
func (s *MemoryStore) CreatePod(pod *api.Pod) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	key := pod.Namespace + "/" + pod.Name
	if _, exists := s.pods[key]; exists {
		return errors.New("pod zaten mevcut")
	}

	// Yeni oluşturulan pod ilk olarak 'Pending' durumuna geçer
	pod.Phase = api.PodPending
	s.pods[key] = pod
	return nil
}

// Tüm pod'ları listeleme
func (s *MemoryStore) ListPods() []*api.Pod {
	s.mu.RLock()
	defer s.mu.RUnlock()

	list := make([]*api.Pod, 0, len(s.pods))
	for _, p := range s.pods {
		list = append(list, p)
	}
	return list
}

// Pod güncelleme (Scheduler node atadığında ya da Kubelet durum değiştirdiğinde)
func (s *MemoryStore) UpdatePod(pod *api.Pod) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	key := pod.Namespace + "/" + pod.Name
	if _, exists := s.pods[key]; !exists {
		return errors.New("güncellenecek pod bulunamadı")
	}

	s.pods[key] = pod
	return nil
}

// Pod silme (Gerçek K8s tarzı Soft-Delete)
func (s *MemoryStore) DeletePod(namespace, name string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	key := namespace + "/" + name
	pod, exists := s.pods[key]
	if !exists {
		return errors.New("silinecek pod bulunamadı")
	}

	// Pod'u hemen map'ten silmiyoruz!
	// Kubelet'in haberi olsun ve container'ı durdurabilsin diye Terminating yapıyoruz.
	now := time.Now()
	pod.DeletionTimestamp = &now
	pod.Phase = api.PodTerminating
	return nil
}

// Kubelet temizliğini bitirdiğinde pod'u tamamen hafızadan kaldıran fonksiyon
func (s *MemoryStore) HardDeletePod(namespace, name string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	key := namespace + "/" + name
	if _, exists := s.pods[key]; !exists {
		return errors.New("pod bulunamadı")
	}

	delete(s.pods, key)
	return nil
}

// ------------------- NODE İŞLEMLERİ -------------------

// Kümeye yeni node kaydetme
func (s *MemoryStore) RegisterNode(node *api.Node) {
	s.mu.Lock()
	defer s.mu.Unlock()

	node.Status = "Ready"
	s.nodes[node.Name] = node
}

// Mevcut node'ları listeleme
func (s *MemoryStore) ListNodes() []*api.Node {
	s.mu.RLock()
	defer s.mu.RUnlock()

	list := make([]*api.Node, 0, len(s.nodes))
	for _, n := range s.nodes {
		list = append(list, n)
	}
	return list
}
