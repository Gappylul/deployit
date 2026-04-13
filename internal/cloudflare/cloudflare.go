package cloudflare

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
)

type Client struct {
	token     string
	accountID string
	tunnelID  string
	zoneID    string
	http      *http.Client
}

func NewClient() (*Client, error) {
	token := os.Getenv("CLOUDFLARE_TOKEN")
	accountID := os.Getenv("CLOUDFLARE_ACCOUNT_ID")
	tunnelID := os.Getenv("CLOUDFLARE_TUNNEL_ID")
	zoneID := os.Getenv("CLOUDFLARE_ZONE_ID")

	if token == "" || accountID == "" || tunnelID == "" || zoneID == "" {
		return nil, fmt.Errorf("CLOUDFLARE_TOKEN, CLOUDFLARE_ACCOUNT_ID, CLOUDFLARE_TUNNEL_ID, and CLOUDFLARE_ZONE_ID must be set")
	}

	return &Client{
		token:     token,
		accountID: accountID,
		tunnelID:  tunnelID,
		zoneID:    zoneID,
		http:      &http.Client{},
	}, nil
}

func (c *Client) AddHostname(hostname string) error {
	if err := c.addTunnelRoute(hostname); err != nil {
		return fmt.Errorf("add tunnel route: %w", err)
	}
	if err := c.addDNSRecord(hostname); err != nil {
		return fmt.Errorf("add dns record: %w", err)
	}
	fmt.Printf("→ cloudflare: routed %s through tunnel\n", hostname)
	return nil
}

func (c *Client) RemoveHostname(hostname string) error {
	if err := c.removeTunnelRoute(hostname); err != nil {
		return fmt.Errorf("remove tunnel route: %w", err)
	}
	if err := c.removeDNSRecord(hostname); err != nil {
		return fmt.Errorf("remove dns record: %w", err)
	}
	fmt.Printf("-> cloudflare: removed %s from tunnel\n", hostname)
	return nil
}

type tunnelConfig struct {
	Config struct {
		Ingress []ingressRule `json:"ingress"`
	} `json:"config"`
}

type ingressRule struct {
	Hostname string `json:"hostname,omitempty"`
	Service  string `json:"service"`
}

func (c *Client) getTunnelConfig() ([]ingressRule, error) {
	url := fmt.Sprintf("https://api.cloudflare.com/client/v4/accounts/%s/cfd_tunnel/%s/configurations",
		c.accountID, c.tunnelID)

	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result struct {
		Result tunnelConfig `json:"result"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return result.Result.Config.Ingress, nil
}

func (c *Client) putTunnelConfig(ingress []ingressRule) error {
	url := fmt.Sprintf("https://api.cloudflare.com/client/v4/accounts/%s/cfd_tunnel/%s/configurations",
		c.accountID, c.tunnelID)

	body, _ := json.Marshal(map[string]interface{}{
		"config": map[string]interface{}{
			"ingress": ingress,
		},
	})

	req, _ := http.NewRequest("PUT", url, bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("cloudflare API returned %d", resp.StatusCode)
	}
	return nil
}

func (c *Client) addTunnelRoute(hostname string) error {
	ingress, err := c.getTunnelConfig()
	if err != nil {
		return err
	}

	for _, rule := range ingress {
		if rule.Hostname == hostname {
			return nil
		}
	}

	catchAll := ingressRule{Service: "http_status:404"}
	var filtered []ingressRule
	for _, rule := range ingress {
		if rule.Hostname == "" {
			catchAll = rule
		} else {
			filtered = append(filtered, rule)
		}
	}

	newIngress := append(filtered, ingressRule{
		Hostname: hostname,
		Service:  "http://192.168.1.214:80",
	}, catchAll)

	return c.putTunnelConfig(newIngress)
}

func (c *Client) removeTunnelRoute(hostname string) error {
	ingress, err := c.getTunnelConfig()
	if err != nil {
		return err
	}

	var filtered []ingressRule
	for _, rule := range ingress {
		if rule.Hostname != hostname {
			filtered = append(filtered, rule)
		}
	}

	return c.putTunnelConfig(filtered)
}

func (c *Client) addDNSRecord(hostname string) error {
	url := fmt.Sprintf("https://api.cloudflare.com/client/v4/zones/%s/dns_records", c.zoneID)

	body, _ := json.Marshal(map[string]interface{}{
		"type":    "CNAME",
		"name":    hostname,
		"content": c.tunnelID + ".cfargotunnel.com",
		"proxied": true,
	})

	req, _ := http.NewRequest("POST", url, bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusBadRequest {
		var result struct {
			Errors []struct {
				Code int `json:"code"`
			} `json:"errors"`
		}
		json.NewDecoder(resp.Body).Decode(&result)
		for _, e := range result.Errors {
			if e.Code == 81053 {
				return nil
			}
		}
		return fmt.Errorf("cloudflare DNS API returned %d", resp.StatusCode)
	}

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("cloudflare DNS API returned %d", resp.StatusCode)
	}
	return nil
}

func (c *Client) removeDNSRecord(hostname string) error {
	url := fmt.Sprintf("https://api.cloudflare.com/client/v4/zones/%s/dns_records?name=%s", c.zoneID, hostname)

	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	var result struct {
		Result []struct {
			ID string `json:"id"`
		} `json:"result"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return err
	}

	if len(result.Result) == 0 {
		return nil
	}

	recordID := result.Result[0].ID
	delURL := fmt.Sprintf("https://api.cloudflare.com/client/v4/zones/%s/dns_records/%s", c.zoneID, recordID)

	req, _ = http.NewRequest("DELETE", delURL, nil)
	req.Header.Set("Authorization", "Bearer "+c.token)

	resp, err = c.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return nil
}
