package readonlyplugin

import (
	"context"
	"testing"

	"github.com/wmentor/go-magnetar/internal/plugin"
)

func TestExecute(t *testing.T) {
	t.Parallel()

	lamp := &plugin.State{
		ReadOnly: false,
	}

	cmd := &Plugin{
		state: lamp,
	}

	params := &plugin.ChatCommandInput{
		Args: "on",
	}

	resp, err := cmd.execute(context.Background(), params)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if resp == nil {
		t.Fatalf("Expected response, got nil")
	}

	if lamp.ReadOnly != true {
		t.Fatalf("Expected ReadOnly to be true, got %v", lamp.ReadOnly)
	}

	if resp.Stdout != "Read-only mode enabled.\n" {
		t.Fatalf("Expected response stdout to be 'Read-only mode enabled.\n', got %v", resp.Stdout)
	}
}

func TestExecuteOff(t *testing.T) {
	t.Parallel()

	lamp := &plugin.State{
		ReadOnly: true,
	}

	cmd := &Plugin{
		state: lamp,
	}

	params := &plugin.ChatCommandInput{
		Args: "off",
	}

	resp, err := cmd.execute(context.Background(), params)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if resp == nil {
		t.Fatalf("Expected response, got nil")
	}

	if lamp.ReadOnly != false {
		t.Fatalf("Expected ReadOnly to be false, got %v", lamp.ReadOnly)
	}

	if resp.Stdout != "Read-only mode disabled.\n" {
		t.Fatalf("Expected response stdout to be 'Read-only mode disabled.\n', got %v", resp.Stdout)
	}
}

func TestExecuteStatusOn(t *testing.T) {
	t.Parallel()

	lamp := &plugin.State{
		ReadOnly: true,
	}

	cmd := &Plugin{
		state: lamp,
	}

	params := &plugin.ChatCommandInput{
		Args: "",
	}

	resp, err := cmd.execute(context.Background(), params)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if resp == nil {
		t.Fatalf("Expected response, got nil")
	}

	if lamp.ReadOnly != true {
		t.Fatalf("Expected ReadOnly to be true, got %v", lamp.ReadOnly)
	}

	if resp.Stdout != "Read-only mode is currently ON.\n" {
		t.Fatalf("Expected response stdout to be 'Read-only mode is currently ON.\n', got %v", resp.Stdout)
	}
}

func TestExecuteStatusOff(t *testing.T) {
	t.Parallel()

	lamp := &plugin.State{
		ReadOnly: false,
	}

	cmd := &Plugin{
		state: lamp,
	}

	params := &plugin.ChatCommandInput{
		Args: "",
	}

	resp, err := cmd.execute(context.Background(), params)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if resp == nil {
		t.Fatalf("Expected response, got nil")
	}

	if lamp.ReadOnly != false {
		t.Fatalf("Expected ReadOnly to be false, got %v", lamp.ReadOnly)
	}

	if resp.Stdout != "Read-only mode is currently OFF.\n" {
		t.Fatalf("Expected response stdout to be 'Read-only mode is currently OFF.\n', got %v", resp.Stdout)
	}
}
