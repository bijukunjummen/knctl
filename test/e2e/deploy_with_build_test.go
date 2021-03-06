/*
Copyright 2018 The Knative Authors

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

package e2e

import (
	"strings"
	"testing"
)

func TestDeployWithBuildPublicImage(t *testing.T) {
	logger := Logger{}
	env := BuildEnv(t)
	knctl := Knctl{t, env.Namespace, logger}
	kubectl := Kubectl{t, env.Namespace, logger}
	curl := Curl{t, knctl}

	const (
		serviceName              = "test-d-w-b-pub-i-service-name"
		pushPullDockerSecretName = serviceName + "-docker-secret"
		buildServiceAccountName  = serviceName + "-service-account"
		expectedContent1         = "TestDeployWithBuild_ContentV1"
		expectedContent2         = "TestDeployWithBuild_ContentV2"
	)

	cleanUp := func() {
		knctl.RunWithOpts([]string{"delete", "service", "-s", serviceName}, RunOpts{AllowError: true})
		kubectl.RunWithOpts([]string{"delete", "secret", pushPullDockerSecretName}, RunOpts{AllowError: true})
		kubectl.RunWithOpts([]string{"delete", "serviceaccount", buildServiceAccountName}, RunOpts{AllowError: true})
	}

	logger.Section("Delete previous service with the same name if exists", cleanUp)
	defer cleanUp()

	logger.Section("Add service account with Docker push secret", func() {
		knctl.RunWithOpts([]string{
			"create",
			"basic-auth-secret",
			"-s", pushPullDockerSecretName,
			"--docker-hub",
			"-u", env.BuildDockerUsername,
			"-p", env.BuildDockerPassword,
		}, RunOpts{Redact: true})

		knctl.Run([]string{"create", "service-account", "-a", buildServiceAccountName, "-s", pushPullDockerSecretName})
	})

	logger.Section("Deploy service v1", func() {
		knctl.Run([]string{
			"deploy",
			"-s", serviceName,
			"--git-url", env.BuildGitURL,
			"--git-revision", env.BuildGitRevisionV1,
			"-i", env.BuildPublicImage,
			"--service-account", buildServiceAccountName,
			"-e", "SIMPLE_MSG=" + expectedContent1,
		})
	})

	logger.Section("Checking if service is reachable and presents content", func() {
		curl.WaitForContent(serviceName, expectedContent1)
	})

	logger.Section("Deploy service v2 with a Git change (new env variable)", func() {
		knctl.Run([]string{
			"deploy",
			"-s", serviceName,
			"--git-url", env.BuildGitURL,
			"--git-revision", env.BuildGitRevisionV2,
			"-i", env.BuildPublicImage,
			"--service-account", buildServiceAccountName,
			"-e", "SIMPLE_MSG_V2=" + expectedContent2,
		})
	})

	logger.Section("Checking if service is reachable and presents content", func() {
		curl.WaitForContent(serviceName, expectedContent2)
	})

	logger.Section("Deleting service", func() {
		knctl.Run([]string{"delete", "service", "-s", serviceName})

		out := knctl.Run([]string{"list", "services", "--json"})
		if strings.Contains(out, serviceName) {
			t.Fatalf("Expected to not see sample service in the list of services, but was: %s", out)
		}
	})
}

func TestDeployWithBuildPrivateImage(t *testing.T) {
	logger := Logger{}
	env := BuildEnv(t)
	knctl := Knctl{t, env.Namespace, logger}
	kubectl := Kubectl{t, env.Namespace, logger}
	curl := Curl{t, knctl}

	const (
		serviceName              = "test-d-w-b-priv-i-service-name"
		pushPullDockerSecretName = serviceName + "-docker-secret"
		pullDockerSecretName     = serviceName + "-p-docker-secret"
		buildServiceAccountName  = serviceName + "-service-account"
		expectedContent1         = "TestDeployWithBuild_ContentV1"
		expectedContent2         = "TestDeployWithBuild_ContentV2"
	)

	cleanUp := func() {
		knctl.RunWithOpts([]string{"delete", "service", "-s", serviceName}, RunOpts{AllowError: true})
		kubectl.RunWithOpts([]string{"delete", "secret", pushPullDockerSecretName}, RunOpts{AllowError: true})
		kubectl.RunWithOpts([]string{"delete", "secret", pullDockerSecretName}, RunOpts{AllowError: true})
		kubectl.RunWithOpts([]string{"delete", "serviceaccount", buildServiceAccountName}, RunOpts{AllowError: true})
	}

	logger.Section("Delete previous service with the same name if exists", cleanUp)
	defer cleanUp()

	logger.Section("Add service account with Docker push secret", func() {
		knctl.RunWithOpts([]string{
			"create",
			"basic-auth-secret",
			"-s", pushPullDockerSecretName,
			"--docker-hub",
			"-u", env.BuildDockerUsername,
			"-p", env.BuildDockerPassword,
		}, RunOpts{Redact: true})

		kubectl.RunWithOpts([]string{
			"create", "secret", "docker-registry", pullDockerSecretName,
			"--docker-server", "https://index.docker.io",
			"--docker-username", env.BuildDockerUsername,
			"--docker-password", env.BuildDockerPassword,
			"--docker-email", "foo",
		}, RunOpts{Redact: true})

		knctl.Run([]string{
			"create",
			"service-account",
			"-a", buildServiceAccountName,
			"-s", pushPullDockerSecretName,
			"-p", pullDockerSecretName,
		})
	})

	logger.Section("Deploy service v1", func() {
		knctl.Run([]string{
			"deploy",
			"-s", serviceName,
			"--git-url", env.BuildGitURL,
			"--git-revision", env.BuildGitRevisionV1,
			"-i", env.BuildPrivateImage,
			"--service-account", buildServiceAccountName,
			"-e", "SIMPLE_MSG=" + expectedContent1,
		})
	})

	logger.Section("Checking if service is reachable and presents content", func() {
		curl.WaitForContent(serviceName, expectedContent1)
	})

	logger.Section("Deploy service v2 with a Git change (new env variable)", func() {
		knctl.Run([]string{
			"deploy",
			"-s", serviceName,
			"--git-url", env.BuildGitURL,
			"--git-revision", env.BuildGitRevisionV2,
			"-i", env.BuildPrivateImage,
			"--service-account", buildServiceAccountName,
			"-e", "SIMPLE_MSG_V2=" + expectedContent2,
		})
	})

	logger.Section("Checking if service is reachable and presents content", func() {
		curl.WaitForContent(serviceName, expectedContent2)
	})

	logger.Section("Deleting service", func() {
		knctl.Run([]string{"delete", "service", "-s", serviceName})

		out := knctl.Run([]string{"list", "services", "--json"})
		if strings.Contains(out, serviceName) {
			t.Fatalf("Expected to not see sample service in the list of services, but was: %s", out)
		}
	})
}
