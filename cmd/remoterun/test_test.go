// Copyright 2024 The Okteto Authors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package remoterun

import (
	"fmt"
	"testing"

	"github.com/okteto/okteto/pkg/deployable"
	"github.com/okteto/okteto/pkg/okteto"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type fakeTestRunner struct {
	mock.Mock
}

func (f *fakeTestRunner) RunTest(params deployable.TestParameters) error {
	args := f.Called(params)
	return args.Error(0)
}

func TestRun_TestCommand(t *testing.T) {
	tests := []struct {
		name     string
		expected error
		params   deployable.TestParameters
	}{
		{
			name: "WithoutError",
			params: deployable.TestParameters{
				Name:      "test-destroy",
				Namespace: "test-namespace",
			},
			expected: nil,
		},
		{
			name: "WithError",
			params: deployable.TestParameters{
				Name:      "test-destroy",
				Namespace: "test-namespace",
			},
			expected: fmt.Errorf("boooooom"),
		},
	}

	okteto.CurrentStore = &okteto.ContextStore{
		CurrentContext: "test",
		Contexts: map[string]*okteto.Context{
			"test": {
				Token: "token",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			runner := &fakeTestRunner{}
			runner.On("RunTest", tt.params).Return(tt.expected)
			err := runner.RunTest(tt.params)

			require.Equal(t, err, tt.expected)
			runner.AssertExpectations(t)
		})
	}

}
