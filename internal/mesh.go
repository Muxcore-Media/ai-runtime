package internal

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/Muxcore-Media/contracts-ai/infer"
	meshv1 "github.com/Muxcore-Media/core/proto/gen/muxcore/mesh/v1"
	modulesdk "github.com/Muxcore-Media/core/sdk/go/module"
	"google.golang.org/grpc"
)

const (
	meshComplete   = "Complete"
	meshEmbed      = "Embed"
	meshTranscribe = "Transcribe"
	meshClassify   = "Classify"
	meshTranslate  = "Translate"
	meshProviders  = "Providers"
)

type aiMeshServer struct {
	meshv1.UnimplementedModuleMeshServer
	moduleID string
	settings modulesdk.SettingsHandler
	m        *Module
}

func registerAIMesh(srv *grpc.Server, moduleID string, m *Module) {
	meshv1.RegisterModuleMeshServer(srv, &aiMeshServer{
		moduleID: moduleID,
		settings: modulesdk.SettingsHandlerFromProvider(m),
		m:        m,
	})
}

func (s *aiMeshServer) Call(ctx context.Context, req *meshv1.CallRequest) (*meshv1.CallResponse, error) {
	if req.GetTargetModule() != "" && req.GetTargetModule() != s.moduleID {
		return &meshv1.CallResponse{Error: fmt.Sprintf("wrong target module %q", req.GetTargetModule())}, nil
	}
	switch req.GetMethod() {
	case "Settings":
		raw, err := json.Marshal(s.settings.List())
		if err != nil {
			return &meshv1.CallResponse{Error: err.Error()}, nil
		}
		return &meshv1.CallResponse{Payload: raw}, nil
	case "UpdateSetting":
		var body struct {
			Key, Value string
		}
		if err := json.Unmarshal(req.GetPayload(), &body); err != nil {
			return &meshv1.CallResponse{Error: err.Error()}, nil
		}
		if err := s.settings.Update(body.Key, body.Value); err != nil {
			return &meshv1.CallResponse{Error: err.Error()}, nil
		}
		return &meshv1.CallResponse{Payload: []byte(`{"ok":true}`)}, nil
	case meshProviders:
		return marshalMesh(map[string]any{"providers": []string{infer.ProviderHeuristic}})
	case meshComplete:
		var body infer.CompleteRequest
		if err := json.Unmarshal(req.GetPayload(), &body); err != nil {
			return &meshv1.CallResponse{Error: err.Error()}, nil
		}
		resp, err := s.m.provider(body.Provider).Complete(ctx, body)
		return marshalMeshErr(resp, err)
	case meshEmbed:
		var body infer.EmbedRequest
		if err := json.Unmarshal(req.GetPayload(), &body); err != nil {
			return &meshv1.CallResponse{Error: err.Error()}, nil
		}
		resp, err := s.m.provider(body.Provider).Embed(ctx, body)
		return marshalMeshErr(resp, err)
	case meshTranscribe:
		var body infer.TranscribeRequest
		if err := json.Unmarshal(req.GetPayload(), &body); err != nil {
			return &meshv1.CallResponse{Error: err.Error()}, nil
		}
		resp, err := s.m.provider(body.Provider).Transcribe(ctx, body)
		return marshalMeshErr(resp, err)
	case meshClassify:
		var body infer.ClassifyRequest
		if err := json.Unmarshal(req.GetPayload(), &body); err != nil {
			return &meshv1.CallResponse{Error: err.Error()}, nil
		}
		resp, err := s.m.provider(body.Provider).Classify(ctx, body)
		return marshalMeshErr(resp, err)
	case meshTranslate:
		var body infer.TranslateRequest
		if err := json.Unmarshal(req.GetPayload(), &body); err != nil {
			return &meshv1.CallResponse{Error: err.Error()}, nil
		}
		resp, err := s.m.provider(body.Provider).Translate(ctx, body)
		return marshalMeshErr(resp, err)
	default:
		return &meshv1.CallResponse{Error: fmt.Sprintf("unknown method %q", req.GetMethod())}, nil
	}
}

func (s *aiMeshServer) StreamCall(meshv1.ModuleMesh_StreamCallServer) error {
	return fmt.Errorf("StreamCall not supported")
}

func marshalMesh(v any) (*meshv1.CallResponse, error) {
	raw, err := json.Marshal(v)
	if err != nil {
		return &meshv1.CallResponse{Error: err.Error()}, nil
	}
	return &meshv1.CallResponse{Payload: raw}, nil
}

func marshalMeshErr(v any, err error) (*meshv1.CallResponse, error) {
	if err != nil {
		return &meshv1.CallResponse{Error: err.Error()}, nil
	}
	return marshalMesh(v)
}
