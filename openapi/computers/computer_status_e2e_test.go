//go:build e2e

package computers_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/yaoapp/yao/e2e/client"
	"github.com/yaoapp/yao/openapi"
	"github.com/yaoapp/yao/openapi/oauth"
	userpkg "github.com/yaoapp/yao/openapi/user"
)

func TestE2EComputerStatusAndEvents(t *testing.T) {
	environment := testEnvironment(t)
	service, ok := openapi.Server.OAuth.(*oauth.Service)
	if !ok || service == nil {
		t.Fatal("OAuth service is not available")
	}
	ctx := context.Background()
	provider, err := service.GetUserProvider()
	if err != nil {
		t.Fatalf("get user provider: %v", err)
	}
	admin, err := provider.GetUserByEmail(ctx, "admin@e2e.local")
	if err != nil {
		t.Fatalf("get admin: %v", err)
	}
	adminLogin, err := userpkg.LoginByUserID(fmt.Sprint(admin["user_id"]), nil)
	if err != nil {
		t.Fatalf("login admin: %v", err)
	}
	adminClient := client.New(environment.BaseURL)
	adminClient.Token = adminLogin.AccessToken

	// Desktop: exchange the bearer session for a short-lived WS ticket.
	resp, body, err := adminClient.PostJSON("/v1/events/ticket", nil)
	if err != nil || resp.StatusCode != http.StatusOK {
		t.Fatalf("ticket status=%v err=%v body=%s", resp, err, body)
	}
	var ticketBody struct {
		Ticket    string `json:"ticket"`
		ExpiresIn int    `json:"expires_in"`
	}
	if err := json.Unmarshal(body, &ticketBody); err != nil || ticketBody.Ticket == "" || ticketBody.ExpiresIn <= 0 {
		t.Fatalf("unexpected ticket body: %s err=%v", body, err)
	}
	adminWS := dialEventWS(t, environment.BaseURL, ticketBody.Ticket)
	defer adminWS.Close()

	connResp, connBody, err := adminClient.PostJSON("/v1/computers/connect", map[string]interface{}{
		"name": "Status Event E2E", "data_dir": "/tmp/e2e-status-event", "visibility": "private",
	})
	if err != nil || connResp.StatusCode != http.StatusOK {
		t.Fatalf("connect status=%v err=%v body=%s", connResp, err, connBody)
	}
	var created connectResponse
	if err := json.Unmarshal(connBody, &created); err != nil {
		t.Fatalf("decode connect: %v", err)
	}
	t.Cleanup(func() { _ = service.RevokeNodeTokenByComputer(context.Background(), created.ComputerID) })
	createdEvent := readComputerEvent(t, adminWS, created.ComputerID)
	if createdEvent["status"] != "pending" {
		t.Fatalf("unexpected created event: %v", createdEvent)
	}

	// Lightweight status endpoint, no full detail.
	resp, body, err = adminClient.Get("/v1/computers/" + created.ComputerID + "/status")
	if err != nil || resp.StatusCode != http.StatusOK {
		t.Fatalf("admin status=%v err=%v body=%s", resp, err, body)
	}
	var status map[string]interface{}
	if err := json.Unmarshal(body, &status); err != nil {
		t.Fatalf("decode status: %v", err)
	}
	if status["computer_id"] != created.ComputerID || status["status"] != "pending" || len(status) != 2 {
		t.Fatalf("unexpected status body: %v", status)
	}
	resp, body, err = adminClient.Get("/v1/computers/missing-status/status")
	if err != nil || resp.StatusCode != http.StatusNotFound {
		t.Fatalf("missing status=%v err=%v body=%s", resp, err, body)
	}

	// Status change publishes a computer.status event with a summary payload.
	resp, body, err = adminClient.Do(mustRequest(http.MethodPost, environment.BaseURL+"/v1/computers/"+created.ComputerID+"/disconnect", ""))
	if err != nil || resp.StatusCode != http.StatusOK {
		t.Fatalf("disconnect status=%v err=%v body=%s", resp, err, body)
	}
	event := readComputerEvent(t, adminWS, created.ComputerID)
	if event["status"] != "disconnected" {
		t.Fatalf("unexpected event: %v", event)
	}
	for key := range event {
		if strings.HasPrefix(key, "__yao_") || key == "node_token" || key == "metadata" {
			t.Fatalf("event leaked internal field %s: %v", key, event)
		}
	}

	// A different user must not receive the owner's computer event.
	normal, err := provider.GetUserByEmail(ctx, "user@e2e.local")
	if err != nil {
		t.Fatalf("get normal user: %v", err)
	}
	normalLogin, err := userpkg.LoginByUserID(fmt.Sprint(normal["user_id"]), nil)
	if err != nil {
		t.Fatalf("login normal: %v", err)
	}
	normalClient := client.New(environment.BaseURL)
	normalClient.Token = normalLogin.AccessToken
	resp, body, err = normalClient.Get("/v1/computers/" + created.ComputerID + "/status")
	if err != nil || resp.StatusCode != http.StatusForbidden {
		t.Fatalf("normal status=%v err=%v body=%s", resp, err, body)
	}
	resp, body, err = normalClient.PostJSON("/v1/events/ticket", nil)
	if err != nil || resp.StatusCode != http.StatusOK {
		t.Fatalf("normal ticket status=%v err=%v body=%s", resp, err, body)
	}
	var normalTicket struct {
		Ticket string `json:"ticket"`
	}
	if err := json.Unmarshal(body, &normalTicket); err != nil || normalTicket.Ticket == "" {
		t.Fatalf("normal ticket body: %s", body)
	}
	normalWS := dialEventWS(t, environment.BaseURL, normalTicket.Ticket)
	defer normalWS.Close()
	_, _, _ = adminClient.Do(mustRequest(http.MethodPost, environment.BaseURL+"/v1/computers/"+created.ComputerID+"/reconnect", ""))
	if got := readEventWithin(normalWS, 600*time.Millisecond); got != nil && got["computer_id"] == created.ComputerID {
		t.Fatalf("normal user received owner computer event: %v", got)
	}
}

func dialEventWS(t *testing.T, baseURL, ticket string) *websocket.Conn {
	t.Helper()
	wsURL := strings.Replace(baseURL, "http://", "ws://", 1) + "/v1/events?ticket=" + url.QueryEscape(ticket)
	conn, resp, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		status := 0
		if resp != nil {
			status = resp.StatusCode
		}
		t.Fatalf("dial events ws status=%d err=%v", status, err)
	}
	return conn
}

func readComputerEvent(t *testing.T, conn *websocket.Conn, computerID string) map[string]interface{} {
	t.Helper()
	if got := readEventWithin(conn, 3*time.Second); got != nil && got["computer_id"] == computerID {
		return got
	}
	t.Fatalf("did not receive computer event for %s", computerID)
	return nil
}

func readEventWithin(conn *websocket.Conn, timeout time.Duration) map[string]interface{} {
	_ = conn.SetReadDeadline(time.Now().Add(timeout))
	for {
		var msg struct {
			Type string                 `json:"type"`
			Data map[string]interface{} `json:"data"`
		}
		if err := conn.ReadJSON(&msg); err != nil {
			return nil
		}
		if msg.Type == "ping" {
			continue
		}
		if strings.HasPrefix(msg.Type, "computer.") {
			return msg.Data
		}
	}
}
