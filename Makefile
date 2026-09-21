build:
	go build -o bin/apiserver.exe ./cmd/apiserver
	go build -o bin/scheduler.exe ./cmd/scheduler
	go build -o bin/kubelet.exe ./cmd/kubelet
	go build -o bin/kubectl-lite.exe ./cmd/kubectl-lite

apiserver:
	go run ./cmd/apiserver/main.go

scheduler:
	go run ./cmd/scheduler/main.go

kubelet-1:
	go run ./cmd/kubelet/main.go -node=node-1 -addr=localhost:10250

kubelet-2:
	go run ./cmd/kubelet/main.go -node=node-2 -addr=localhost:10251