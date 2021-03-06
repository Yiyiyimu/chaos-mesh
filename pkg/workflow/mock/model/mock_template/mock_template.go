// Copyright Chaos Mesh Authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// See the License for the specific language governing permissions and
// limitations under the License.

// Code generated by MockGen. DO NOT EDIT.
// Source: ./model/template/template.go

// Package mock_template is a generated GoMock package.
package mock_template

import (
	reflect "reflect"

	gomock "github.com/golang/mock/gomock"

	template "github.com/chaos-mesh/chaos-mesh/pkg/workflow/model/template"
)

// MockTemplate is a mock of Template interface.
type MockTemplate struct {
	ctrl     *gomock.Controller
	recorder *MockTemplateMockRecorder
}

// MockTemplateMockRecorder is the mock recorder for MockTemplate.
type MockTemplateMockRecorder struct {
	mock *MockTemplate
}

// NewMockTemplate creates a new mock instance.
func NewMockTemplate(ctrl *gomock.Controller) *MockTemplate {
	mock := &MockTemplate{ctrl: ctrl}
	mock.recorder = &MockTemplateMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockTemplate) EXPECT() *MockTemplateMockRecorder {
	return m.recorder
}

// Name mocks base method.
func (m *MockTemplate) Name() string {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Name")
	ret0, _ := ret[0].(string)
	return ret0
}

// Name indicates an expected call of Name.
func (mr *MockTemplateMockRecorder) Name() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Name", reflect.TypeOf((*MockTemplate)(nil).Name))
}

// TemplateType mocks base method.
func (m *MockTemplate) TemplateType() template.TemplateType {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "TemplateType")
	ret0, _ := ret[0].(template.TemplateType)
	return ret0
}

// TemplateType indicates an expected call of TemplateType.
func (mr *MockTemplateMockRecorder) TemplateType() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "TemplateType", reflect.TypeOf((*MockTemplate)(nil).TemplateType))
}
