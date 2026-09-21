package main

import (
	"flag"
	"fmt"
	"os"
	"text/tabwriter"

	"k8s-lite/pkg/api"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		return
	}

	client := api.NewClient("http://localhost:8080")
	action := os.Args[1]

	switch action {
	case "get":
		handleGet(client)
	case "create":
		handleCreate(client)
	case "delete":
		handleDelete(client)
	default:
		printUsage()
	}
}

func handleGet(client *api.Client) {
	if len(os.Args) < 3 {
		fmt.Println("Kullanim: kubectl-lite get [pods|nodes]")
		return
	}
	resource := os.Args[2]

	w := tabwriter.NewWriter(os.Stdout, 0, 8, 2, '\t', 0)

	switch resource {
	case "pods":
		pods, err := client.ListPods()
		if err != nil {
			fmt.Printf("Podlar listelenemedi: %v\n", err)
			return
		}
		fmt.Fprintln(w, "NAME\tIMAGE\tNODE\tSTATUS")
		for _, p := range pods {
			node := p.NodeName
			if node == "" {
				node = "<none>"
			}
			fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", p.Name, p.Image, node, p.Phase)
		}
		w.Flush()

	case "nodes":
		nodes, err := client.ListNodes()
		if err != nil {
			fmt.Printf("Node listesi alinamadi: %v\n", err)
			return
		}
		pods, _ := client.ListPods()

		fmt.Fprintln(w, "NODE\tADDRESS\tSTATUS\tASSIGNED PODS")
		for _, n := range nodes {
			// Bu node'a atanmış pod'ları topla
			assignedPods := []string{}
			for _, p := range pods {
				if p.NodeName == n.Name {
					assignedPods = append(assignedPods, p.Name)
				}
			}

			podListStr := fmt.Sprintf("%v", assignedPods)
			if len(assignedPods) == 0 {
				podListStr = "none"
			}
			fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", n.Name, n.Address, n.Status, podListStr)
		}
		w.Flush()

	default:
		fmt.Printf("Bilinmeyen kaynak: %s (pods veya nodes kullanin)\n", resource)
	}
}

func handleCreate(client *api.Client) {
	if len(os.Args) < 3 || os.Args[2] != "pod" {
		fmt.Println("Kullanim: kubectl-lite create pod -name=<ad> -image=<imaj>")
		return
	}

	createCmd := flag.NewFlagSet("create pod", flag.ExitOnError)
	name := createCmd.String("name", "", "Pod adi")
	image := createCmd.String("image", "", "Container imaji")

	_ = createCmd.Parse(os.Args[3:])

	if *name == "" || *image == "" {
		fmt.Println("Hata: -name ve -image parametreleri zorunludur.")
		return
	}

	pod := &api.Pod{
		Name:      *name,
		Namespace: "default",
		Image:     *image,
	}

	if err := client.CreatePod(pod); err != nil {
		fmt.Printf("Pod olusturulamadi: %v\n", err)
		return
	}
	fmt.Printf("pod/%s created\n", *name)
}

func handleDelete(client *api.Client) {
	if len(os.Args) < 4 || os.Args[2] != "pod" {
		fmt.Println("Kullanim: kubectl-lite delete pod <pod-name>")
		return
	}
	podName := os.Args[3]

	if err := client.DeletePod("default", podName); err != nil {
		fmt.Printf("Pod silinemedi: %v\n", err)
		return
	}
	fmt.Printf("pod \"%s\" deleted (marked for termination)\n", podName)
}

func printUsage() {
	fmt.Println("kubectl-lite komutlari:")
	fmt.Println("  get pods")
	fmt.Println("  get nodes")
	fmt.Println("  create pod -name=<ad> -image=<imaj>")
	fmt.Println("  delete pod <ad>")
}
