// Copyright 2023 The Okteto Authors
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

package manifest

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/okteto/okteto/pkg/env"
	"github.com/okteto/okteto/pkg/filesystem"
	"github.com/okteto/okteto/pkg/linguist"
	oktetoLog "github.com/okteto/okteto/pkg/log"
	"github.com/okteto/okteto/pkg/model"
	"github.com/okteto/okteto/pkg/model/forward"
	apiv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
)

type piWorkload struct {
	Identity string `json:"identity"`
}

// piManifest struct which contains
// an arrays of services, workers, kConsumers, schedulers
type piManifest struct {
	Services   []piWorkload `json:"services"`
	Workers    []piWorkload `json:"workers"`
	KConsumers []piWorkload `json:"kConsumers"`
	Schedulers []piWorkload `json:"schedulers"`
}

// RunGuestyInitV1 initializes a new okteto manifest based on Guesty requirements
func (mc *Command) RunGuestyInitV1(ctx context.Context, opts *InitOpts) error {
	//getting entity name by current folder name
	entityName := filepath.Base(opts.Workdir)

	// Open menifest json from platform inventory folder
	piManifestByte, err := os.ReadFile(fmt.Sprintf("../platform-inventory/dimensions/guesty/entity/%s:manifest.json", entityName))
	if err != nil {
		oktetoLog.Error(err)
		return err
	}

	oktetoLog.Success(fmt.Sprintf("Opened ../platform-inventory/dimensions/guesty/entity/%s:manifest.json", entityName))

	// we initialize our piManifest array
	var piManifest piManifest

	// we unmarshal our byteArray which contains our
	// piManifestFile's content into 'piManifest' which we defined above
	json.Unmarshal(piManifestByte, &piManifest)

	//Create new empty okteto manifest
	manifest := model.NewManifest()
	//Create new empty okteto manifest dev section
	devs := model.ManifestDevs{}

	manifest.Deploy = nil

	//Loop over PiManifest
	for i := 0; i < len(piManifest.Services) && len(piManifest.Services) != 0; i++ {
		devs[piManifest.Services[i].Identity] = generateGuestyDev(piManifest.Services[i].Identity, "service")
		//identitiesList = append(identitiesList, piManifest.Services[i].Identity)
	}

	for i := 0; i < len(piManifest.Workers) && len(piManifest.Workers) != 0; i++ {
		devs[piManifest.Workers[i].Identity] = generateGuestyDev(piManifest.Workers[i].Identity, "worker")
		//identitiesList = append(identitiesList, piManifest.Services[i].Identity)
	}

	for i := 0; i < len(piManifest.KConsumers) && len(piManifest.KConsumers) != 0; i++ {
		devs[piManifest.KConsumers[i].Identity] = generateGuestyDev(piManifest.KConsumers[i].Identity, "kconsumer")
		//identitiesList = append(identitiesList, piManifest.Services[i].Identity)
	}

	for i := 0; i < len(piManifest.Schedulers) && len(piManifest.Schedulers) != 0; i++ {
		devs[piManifest.Schedulers[i].Identity] = generateGuestyDev(piManifest.Schedulers[i].Identity, "scheduler")
		//identitiesList = append(identitiesList, piManifest.Services[i].Identity)
	}

	// Assign generated devs to manifest
	manifest.Dev = devs

	// save manifest to file
	if err := manifest.WriteToFile(opts.DevPath); err != nil {
		return err
	}

	oktetoLog.Success("Okteto manifest (%s) configured successfully", opts.DevPath)

	devDir, err := filepath.Abs(filepath.Dir(opts.DevPath))
	if err != nil {
		return err
	}
	stignore := filepath.Join(devDir, stignoreFile)

	if !filesystem.FileExists(stignore) {
		c := linguist.GetSTIgnore("node")
		if err := os.WriteFile(stignore, c, 0600); err != nil {
			oktetoLog.Infof("failed to write stignore file: %s", err)
		}
	}

	if opts.ShowCTA {
		oktetoLog.Information("Run 'okteto up' to activate your development container")
	}

	return nil
}

func generateGuestyDev(devName string, workloadType string) *model.Dev {
	dev := &model.Dev{
		Container:            devName,
		Workdir:              "/appdev",
		ImagePullPolicy:      "IfNotPresent",
		BootstrapCommand:     "apt update; apt -y --no-install-recommends install python3 procps; ln -sf /app/node_modules ./node_modules; mkdir -p dist",
		PersistentVolumeInfo: &model.PersistentVolumeInfo{Enabled: false},
		Lifecycle:            &model.Lifecycle{PostStart: true, PostStop: true},
		Keda:                 true,
		Command:              model.Command{Values: []string{"bash"}},
		Sync:                 model.Sync{Folders: []model.SyncFolder{model.SyncFolder{LocalPath: ".", RemotePath: "/appdev"}}},
		Environment:          env.Environment{env.Var{Name: "DEBUG_PORT", Value: "9229"}},
		Forward:              []forward.Forward{forward.Forward{Local: 9229, Remote: 9229}, forward.Forward{Local: 3000, Remote: 3000}},
		Resources:            model.ResourceRequirements{Limits: model.ResourceList{"cpu": resource.MustParse("3"), "memory": resource.MustParse("4Gi")}},
		SecurityContext:      &model.SecurityContext{Capabilities: &model.Capabilities{Add: []apiv1.Capability{"fowner", "chown", "setuid", "setgid"}}},
		NodeSelector:         map[string]string{"WorkloadType": workloadType},
		Secrets:              []model.Secret{model.Secret{"$HOME/.npmrc", "/root/.npmrc", 644}},
	}

	return dev
}
