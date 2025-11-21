// /*
// Copyright © 2025 ESO Maintainer Team
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
// */

// 2025
// Copyright External Secrets Inc.
// All Rights Reserved.

// Package api provides workflow API server.
package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/go-logr/logr"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	workflows "github.com/external-secrets/external-secrets/apis/workflows/v1alpha1"
)

// Server is the API server for workflow operations.
type Server struct {
	client client.Client
	log    logr.Logger
	server *http.Server
}

// WorkflowRunRequest is the request body for creating a workflow run.
type WorkflowRunRequest struct {
	// TemplateName is the name of the template to run.
	TemplateName string `json:"templateName"`

	// TemplateNamespace is the namespace of the template.
	// If not specified, the namespace from the URL is used.
	TemplateNamespace string `json:"templateNamespace,omitempty"`

	// Arguments are the values for template parameters.
	// Each argument corresponds to a parameter defined in the template.
	Arguments map[string]any `json:"arguments,omitempty"`
}

// WorkflowRunResponse is the response body for a workflow run request.
type WorkflowRunResponse struct {
	// Name is the name of the created WorkflowRun.
	Name string `json:"name"`

	// Namespace is the namespace of the created WorkflowRun.
	Namespace string `json:"namespace"`

	// Status is the status of the request.
	Status string `json:"status"`

	// Message is a human-readable message.
	Message string `json:"message,omitempty"`
}

// NewServer creates a new API server.
func NewServer(c client.Client, log logr.Logger) *Server {
	return &Server{
		client: c,
		log:    log,
	}
}

// Start starts the API server.
func (s *Server) Start(addr string) error {
	mux := http.NewServeMux()

	// Register API endpoints
	mux.HandleFunc("/api/v1/namespaces/", s.handleNamespacedRequests)
	mux.HandleFunc("/healthz", s.handleHealthz)

	s.server = &http.Server{
		Addr:              addr,
		Handler:           mux,
		ReadHeaderTimeout: 10 * time.Second,
	}

	s.log.Info("Starting API server", "addr", addr)
	return s.server.ListenAndServe()
}

// Stop stops the API server.
func (s *Server) Stop(ctx context.Context) error {
	s.log.Info("Stopping API server")
	return s.server.Shutdown(ctx)
}

// handleHealthz handles health check requests.
func (s *Server) handleHealthz(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
	_, err := w.Write([]byte("ok"))
	if err != nil {
		s.log.Error(err, "Failed to write health check response")
	}
}

// handleNamespacedRequests handles requests to namespaced resources.
func (s *Server) handleNamespacedRequests(w http.ResponseWriter, r *http.Request) {
	// Extract namespace and resource type from the URL
	parts := strings.Split(r.URL.Path, "/")
	if len(parts) < 5 {
		http.Error(w, "Invalid URL path", http.StatusBadRequest)
		return
	}

	namespace := parts[3]
	resourceType := parts[4]

	switch resourceType {
	case "workflowruns":
		s.handleWorkflowRuns(w, r, namespace)
	default:
		http.Error(w, fmt.Sprintf("Unsupported resource type: %s", resourceType), http.StatusBadRequest)
	}
}

// handleWorkflowRuns handles requests to the workflowruns endpoint.
func (s *Server) handleWorkflowRuns(w http.ResponseWriter, r *http.Request, namespace string) {
	switch r.Method {
	case http.MethodPost:
		s.createWorkflowRun(w, r, namespace)
	case http.MethodGet:
		parts := strings.Split(r.URL.Path, "/")
		if len(parts) > 5 {
			s.getWorkflowRun(w, r, namespace, parts[5])
		} else {
			s.listWorkflowRuns(w, r, namespace)
		}
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// createWorkflowRun creates a new WorkflowRun.
func (s *Server) createWorkflowRun(w http.ResponseWriter, r *http.Request, namespace string) {
	// Parse request body
	var req WorkflowRunRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf("Invalid request body: %v", err), http.StatusBadRequest)
		return
	}

	// Validate request
	if req.TemplateName == "" {
		http.Error(w, "Template name is required", http.StatusBadRequest)
		return
	}

	// Use namespace from URL if not specified in request
	templateNamespace := namespace
	if req.TemplateNamespace != "" {
		templateNamespace = req.TemplateNamespace
	}

	// Check if template exists
	template := &workflows.WorkflowTemplate{}
	if err := s.client.Get(r.Context(), types.NamespacedName{
		Name:      req.TemplateName,
		Namespace: templateNamespace,
	}, template); err != nil {
		http.Error(w, fmt.Sprintf("Template not found: %v", err), http.StatusNotFound)
		return
	}

	// Create WorkflowRun
	argumentBytes, err := json.Marshal(req.Arguments)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error marshaling arguments: %v", err), http.StatusBadRequest)
		return
	}

	run := &workflows.WorkflowRun{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: fmt.Sprintf("%s-", req.TemplateName),
			Namespace:    namespace,
			Labels: map[string]string{
				"workflows.external-secrets.io/template": req.TemplateName,
				"workflows.external-secrets.io/api":      "true",
			},
			Annotations: map[string]string{
				"workflows.external-secrets.io/created-at": time.Now().Format(time.RFC3339),
			},
		},
		Spec: workflows.WorkflowRunSpec{
			TemplateRef: workflows.TemplateRef{
				Name:      req.TemplateName,
				Namespace: templateNamespace,
			},
			Arguments: apiextensionsv1.JSON{
				Raw: argumentBytes,
			},
		},
	}

	// Create the WorkflowRun
	if err := s.client.Create(r.Context(), run); err != nil {
		http.Error(w, fmt.Sprintf("Failed to create WorkflowRun: %v", err), http.StatusInternalServerError)
		return
	}

	// Return response
	resp := WorkflowRunResponse{
		Name:      run.Name,
		Namespace: run.Namespace,
		Status:    "created",
		Message:   "WorkflowRun created successfully",
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		s.log.Error(err, "Failed to encode workflow run response")
	}
}

// getWorkflowRun gets a WorkflowRun by name.
func (s *Server) getWorkflowRun(w http.ResponseWriter, r *http.Request, namespace, name string) {
	run := &workflows.WorkflowRun{}
	if err := s.client.Get(r.Context(), types.NamespacedName{
		Name:      name,
		Namespace: namespace,
	}, run); err != nil {
		http.Error(w, fmt.Sprintf("WorkflowRun not found: %v", err), http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(run); err != nil {
		s.log.Error(err, "Failed to encode workflow run")
	}
}

// listWorkflowRuns lists WorkflowRuns in a namespace.
func (s *Server) listWorkflowRuns(w http.ResponseWriter, r *http.Request, namespace string) {
	runList := &workflows.WorkflowRunList{}
	if err := s.client.List(r.Context(), runList, client.InNamespace(namespace)); err != nil {
		http.Error(w, fmt.Sprintf("Failed to list WorkflowRuns: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(runList); err != nil {
		s.log.Error(err, "Failed to encode workflow run list")
	}
}
