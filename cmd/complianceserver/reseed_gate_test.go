package main

import (
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
)

func TestReseedGateWaitsForActiveRequests(t *testing.T) {
	requestStarted := make(chan struct{})
	releaseRequest := make(chan struct{})
	reseedStarted := make(chan struct{})

	handler := newReseedGate(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/Products":
			close(requestStarted)
			<-releaseRequest
		case "/Reseed":
			close(reseedStarted)
		}
		w.WriteHeader(http.StatusOK)
	}))

	requestDone := make(chan struct{})
	go func() {
		handler.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodPost, "/Products", nil))
		close(requestDone)
	}()
	<-requestStarted

	reseedDone := make(chan struct{})
	go func() {
		handler.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodPost, "/Reseed", nil))
		close(reseedDone)
	}()

	select {
	case <-reseedStarted:
		t.Fatal("reseed started while a database request was active")
	case <-time.After(20 * time.Millisecond):
	}

	close(releaseRequest)
	<-requestDone
	<-reseedDone
}

func TestReseedGateBlocksNewRequestsDuringReseed(t *testing.T) {
	reseedStarted := make(chan struct{})
	releaseReseed := make(chan struct{})
	var productRequests atomic.Int32

	handler := newReseedGate(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/Reseed" {
			close(reseedStarted)
			<-releaseReseed
		} else {
			productRequests.Add(1)
		}
		w.WriteHeader(http.StatusOK)
	}))

	reseedDone := make(chan struct{})
	go func() {
		handler.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodPost, "/Reseed", nil))
		close(reseedDone)
	}()
	<-reseedStarted

	requestDone := make(chan struct{})
	go func() {
		handler.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodPost, "/Products", nil))
		close(requestDone)
	}()

	time.Sleep(20 * time.Millisecond)
	if got := productRequests.Load(); got != 0 {
		t.Fatalf("product requests during reseed = %d, want 0", got)
	}

	close(releaseReseed)
	<-reseedDone
	<-requestDone
	if got := productRequests.Load(); got != 1 {
		t.Fatalf("product requests after reseed = %d, want 1", got)
	}
}
