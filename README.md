# k8s-lite (Educational Kubernetes Control Plane in Go)

Kubernetes'in arkasındaki sihir perdesini aralamak ve çekirdek kontrol düzlemi (control plane) mantığını kavramak için Go ile sıfırdan yazılmış minimal bir orkestrasyon simülasyonu.

## Mimari ve Temel Kavramlar

- **apiserver (`cmd/apiserver`):** Sistemin tek giriş kapısı. In-memory thread-safe (`sync.RWMutex`) bir veri deposu üzerinde CRUD ve soft-delete işlemlerini yürütür.
- **scheduler (`cmd/scheduler`):** Sahipsiz (`Pending` aşamasındaki) pod'ları düzenli aralıklarla tarar ve Round-Robin algoritmasıyla uygun node'lara atar (`Scheduled`).
- **kubelet (`cmd/kubelet`):** Her node üzerinde çalışan bağımsız ajan. Kendi üzerine düşen pod'ları başlatır (`Running`) ve silinmek üzere işaretlenen pod'ları temizler (`Terminating` -> `Deleted`).
- **kubectl-lite (`cmd/kubectl-lite`):** Kümeyi yönetmek için REST API ile haberleşen hafif CLI aracı.

## Hızlı Başlangıç

### 1. Derleme
```bash
make build