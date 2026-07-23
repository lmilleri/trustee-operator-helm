/*
Copyright 2026.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package helm

import (
	"fmt"
	"os"
	"time"

	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chart/loader"
	"helm.sh/helm/v3/pkg/release"
	"helm.sh/helm/v3/pkg/storage/driver"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

const (
	DefaultTimeout = 5 * time.Minute
	ChartPath      = "/opt/helm-charts/trustee"
)

type Client struct {
	namespace string
	getter    genericclioptions.RESTClientGetter
}

func NewClient(getter genericclioptions.RESTClientGetter, namespace string) *Client {
	return &Client{
		namespace: namespace,
		getter:    getter,
	}
}

func (c *Client) newActionConfig() (*action.Configuration, error) {
	logger := log.Log.WithName("helm")
	actionConfig := new(action.Configuration)
	err := actionConfig.Init(c.getter, c.namespace, os.Getenv("HELM_DRIVER"), func(format string, v ...interface{}) {
		logger.Info(fmt.Sprintf(format, v...))
	})
	return actionConfig, err
}

func (c *Client) GetRelease(name string) (*release.Release, error) {
	actionConfig, err := c.newActionConfig()
	if err != nil {
		return nil, fmt.Errorf("initializing action config: %w", err)
	}

	get := action.NewGet(actionConfig)
	rel, err := get.Run(name)
	if err != nil {
		if err == driver.ErrReleaseNotFound {
			return nil, nil
		}
		return nil, fmt.Errorf("getting release %s: %w", name, err)
	}
	return rel, nil
}

func (c *Client) InstallOrUpgrade(releaseName string, vals map[string]interface{}) (*release.Release, error) {
	existing, err := c.GetRelease(releaseName)
	if err != nil {
		return nil, err
	}

	if existing == nil {
		return c.install(releaseName, vals)
	}
	return c.upgrade(releaseName, vals)
}

func (c *Client) install(releaseName string, vals map[string]interface{}) (*release.Release, error) {
	actionConfig, err := c.newActionConfig()
	if err != nil {
		return nil, fmt.Errorf("initializing action config: %w", err)
	}

	chartPath := resolveChartPath()
	chart, err := loader.Load(chartPath)
	if err != nil {
		return nil, fmt.Errorf("loading chart from %s: %w", chartPath, err)
	}

	install := action.NewInstall(actionConfig)
	install.ReleaseName = releaseName
	install.Namespace = c.namespace
	install.CreateNamespace = false
	install.Timeout = DefaultTimeout
	install.Wait = true

	rel, err := install.Run(chart, vals)
	if err != nil {
		return nil, fmt.Errorf("installing release %s: %w", releaseName, err)
	}
	return rel, nil
}

func (c *Client) upgrade(releaseName string, vals map[string]interface{}) (*release.Release, error) {
	actionConfig, err := c.newActionConfig()
	if err != nil {
		return nil, fmt.Errorf("initializing action config: %w", err)
	}

	chartPath := resolveChartPath()
	chart, err := loader.Load(chartPath)
	if err != nil {
		return nil, fmt.Errorf("loading chart from %s: %w", chartPath, err)
	}

	upgrade := action.NewUpgrade(actionConfig)
	upgrade.Namespace = c.namespace
	upgrade.Timeout = DefaultTimeout
	upgrade.Wait = true
	upgrade.ReuseValues = false

	rel, err := upgrade.Run(releaseName, chart, vals)
	if err != nil {
		return nil, fmt.Errorf("upgrading release %s: %w", releaseName, err)
	}
	return rel, nil
}

func (c *Client) Uninstall(releaseName string) error {
	existing, err := c.GetRelease(releaseName)
	if err != nil {
		return err
	}
	if existing == nil {
		return nil
	}

	actionConfig, err := c.newActionConfig()
	if err != nil {
		return fmt.Errorf("initializing action config: %w", err)
	}

	uninstall := action.NewUninstall(actionConfig)
	uninstall.Timeout = DefaultTimeout

	_, err = uninstall.Run(releaseName)
	if err != nil {
		return fmt.Errorf("uninstalling release %s: %w", releaseName, err)
	}
	return nil
}

func ChartVersion() (string, error) {
	chartPath := resolveChartPath()
	chart, err := loader.Load(chartPath)
	if err != nil {
		return "", fmt.Errorf("loading chart from %s: %w", chartPath, err)
	}
	return chart.Metadata.Version, nil
}

func resolveChartPath() string {
	if p := os.Getenv("TRUSTEE_CHART_PATH"); p != "" {
		return p
	}
	return ChartPath
}
