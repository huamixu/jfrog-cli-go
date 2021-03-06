package utils

import (
	"github.com/jfrogdev/jfrog-cli-go/jfrog-cli/utils/config"
	"net/http"
	"github.com/jfrogdev/jfrog-cli-go/jfrog-client/artifactory/services/utils/auth"
	"github.com/jfrogdev/jfrog-cli-go/jfrog-client/artifactory/services/utils/auth/cert"
	"github.com/jfrogdev/jfrog-cli-go/jfrog-client/artifactory/httpclient"
	"github.com/jfrogdev/jfrog-cli-go/jfrog-client/artifactory"
	"github.com/jfrogdev/jfrog-cli-go/jfrog-client/artifactory/services"
	clientutils "github.com/jfrogdev/jfrog-cli-go/jfrog-client/artifactory/services/utils"
	"github.com/jfrogdev/jfrog-cli-go/jfrog-cli/utils/cliutils"
	"os"
	"os/exec"
	"github.com/jfrogdev/jfrog-cli-go/jfrog-client/utils/errorutils"
	"errors"
	"path"
	"net/url"
	"runtime"
	"path/filepath"
)

func GetJfrogSecurityDir() (string, error) {
	homeDir, err := config.GetJfrogHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(homeDir, "security"), nil
}

func GetEncryptedPasswordFromArtifactory(artifactoryAuth *auth.ArtifactoryDetails) (string, error) {
	u, err := url.Parse(artifactoryAuth.Url)
	if err != nil {
		return "", err
	}
	u.Path = path.Join(u.Path, "api/security/encryptedPassword")
	httpClientsDetails := artifactoryAuth.CreateArtifactoryHttpClientDetails()
	securityDir, err := GetJfrogSecurityDir()
	if err != nil {
		return "", err
	}
	transport, err := cert.GetTransportWithLoadedCert(securityDir)
	client := httpclient.NewHttpClient(&http.Client{Transport: transport})
	resp, body, _, err := client.SendGet(u.String(), true, httpClientsDetails)
	if err != nil {
		return "", err
	}

	if resp.StatusCode == http.StatusOK {
		return string(body), nil
	}

	if resp.StatusCode == http.StatusConflict {
		message := "\nYour Artifactory server is not configured to encrypt passwords.\n" +
				"You may use \"art config --enc-password=false\""
		return "", errorutils.CheckError(errors.New(message))
	}

	return "", errorutils.CheckError(errors.New("Artifactory response: " + resp.Status))
}

func CreateServiceManager(artDetails *config.ArtifactoryDetails, isDryRun bool) (*artifactory.ArtifactoryServicesManager, error) {
	certPath, err := GetJfrogSecurityDir()
	if err != nil {
		return nil, err
	}
	artAuth, err := artDetails.CreateArtAuthConfig()
	if err != nil {
		return nil, err
	}
	serviceConfig, err := (&artifactory.ArtifactoryServicesConfigBuilder{}).
		SetArtDetails(artAuth).
		SetCertificatesPath(certPath).
		SetDryRun(isDryRun).
		SetLogger(cliutils.CliLogger).
		Build()
	if err != nil {
		return nil, err
	}
	return artifactory.NewArtifactoryService(serviceConfig)
}

func ConvertResultItemArrayToDeleteItemArray(resultItems []clientutils.ResultItem) ([]services.DeleteItem) {
	var deleteItems []services.DeleteItem = make([]services.DeleteItem, len(resultItems))
	for i, item := range resultItems {
		deleteItems[i] = item
	}
	return deleteItems
}

func RunCmd(config CmdConfig) error {
	for k, v := range config.GetEnv() {
		os.Setenv(k, v)
	}

	cmd := config.GetCmd()
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Start()
	if err != nil {
		return errorutils.CheckError(err)
	}
	err = cmd.Wait()
	if err != nil {
		return errorutils.CheckError(err)
	}

	return nil
}

func GetGradleExecPath(useWrapper bool) (string, error) {
	if useWrapper {
		if runtime.GOOS == "windows" {
			return "gradlew.bat", nil
		}
		return "./gradlew", nil
	}
	gradleExec, err := exec.LookPath("gradle")
	if err != nil {
		return "", errorutils.CheckError(err)
	}
	return gradleExec, nil
}

type CmdConfig interface {
	GetCmd() *exec.Cmd
	GetEnv() map[string]string
}