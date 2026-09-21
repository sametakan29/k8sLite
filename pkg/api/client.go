package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

type Client struct {
	baseURL    string
	httpClient *http.Client
}

func NewClient(baseURL string) *Client {
	return &Client{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 5 * time.Second,
		},
	}
}

// Pod'ları listele: GET /pods
func (c *Client) ListPods() ([]*Pod, error) {
	resp, err := c.httpClient.Get(c.baseURL + "/pods")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var pods []*Pod
	if err := json.NewDecoder(resp.Body).Decode(&pods); err != nil {
		return nil, err
	}
	return pods, nil
}

// Yeni Pod oluştur: POST /pods
func (c *Client) CreatePod(pod *Pod) error {
	data, err := json.Marshal(pod)
	if err != nil {
		return err
	}

	resp, err := c.httpClient.Post(c.baseURL+"/pods", "application/json", bytes.NewReader(data))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("pod olusturulamadi: HTTP %d", resp.StatusCode)
	}
	return nil
}

// Pod güncelle: PUT /pods/{ns}/{name}
func (c *Client) UpdatePod(pod *Pod) error {
	data, err := json.Marshal(pod)
	if err != nil {
		return err
	}

	url := fmt.Sprintf("%s/pods/%s/%s", c.baseURL, pod.Namespace, pod.Name)
	req, err := http.NewRequest(http.MethodPut, url, bytes.NewReader(data))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("pod guncellenemedi: HTTP %d", resp.StatusCode)
	}
	return nil
}

// Pod sil: DELETE /pods/{ns}/{name}
func (c *Client) DeletePod(namespace, name string) error {
	url := fmt.Sprintf("%s/pods/%s/%s", c.baseURL, namespace, name)
	req, err := http.NewRequest(http.MethodDelete, url, nil)
	if err != nil {
		return err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("pod silinemedi: HTTP %d", resp.StatusCode)
	}
	return nil
}

// Node kaydet: POST /nodes
func (c *Client) RegisterNode(node *Node) error {
	data, err := json.Marshal(node)
	if err != nil {
		return err
	}

	resp, err := c.httpClient.Post(c.baseURL+"/nodes", "application/json", bytes.NewReader(data))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("node kaydedilemedi: HTTP %d", resp.StatusCode)
	}
	return nil
}

// Node'ları listele: GET /nodes
func (c *Client) ListNodes() ([]*Node, error) {
	resp, err := c.httpClient.Get(c.baseURL + "/nodes")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var nodes []*Node
	if err := json.NewDecoder(resp.Body).Decode(&nodes); err != nil {
		return nil, err
	}
	return nodes, nil
}
