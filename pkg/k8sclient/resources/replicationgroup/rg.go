package replicationgroup

import (
	"cert-csi/pkg/utils"
	"context"
	"fmt"
	replalpha "github.com/dell/csm-replication/api/v1alpha1"
	"github.com/fatih/color"
	"github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	runtimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"strings"
	"time"
)

const (
	// Timeout is a timeout interval for RG actions
	Timeout = 1800 * time.Second
)

// Client is a client for managing RGs
type Client struct {
	Interface runtimeclient.Client
	ClientSet kubernetes.Interface
	Timeout   int
}

type RG struct {
	Client  *Client
	Object  *replalpha.DellCSIReplicationGroup
	Deleted bool

	// Used when error arises in syncable methods
	error error
}

func (c *Client) Delete(ctx context.Context, rg *replalpha.DellCSIReplicationGroup) *RG {
	var funcErr error

	err := c.Interface.Delete(ctx, rg)
	if err != nil {
		funcErr = err
	}
	logrus.Debugf("Deleted RG %s", rg.GetName())
	return &RG{
		Client:  c,
		Object:  rg,
		Deleted: true,
		error:   funcErr,
	}
}

func (c *Client) Get(ctx context.Context, name string) *RG {
	var funcErr error

	rgObject := &replalpha.DellCSIReplicationGroup{}

	err := c.Interface.Get(ctx, types.NamespacedName{Name: name}, rgObject)
	if err != nil {
		funcErr = err
	}

	logrus.Debugf("Got the Rg  %s", rgObject.GetName())
	return &RG{
		Client:  c,
		Object:  rgObject,
		Deleted: false,
		error:   funcErr,
	}
}

func (rg *RG) Name() string {
	return rg.Object.Name
}

func (rg *RG) ExecuteAction(ctx context.Context, rgAction string) error {
	var funcErr error
	log := utils.GetLoggerFromContext(ctx)
	startTime := time.Now()
	timeout := Timeout
	if rg.Client.Timeout != 0 {
		timeout = time.Duration(rg.Client.Timeout) * time.Second
	}

	driverName := rg.Object.Labels["replication.storage.dell.com/driverName"]
	rgName := rg.Object.Name
	rg.Object.Spec.Action = rgAction
	err := rg.Client.Interface.Update(ctx, rg.Object)
	if err != nil {
		funcErr = err
	}

	rgObject := &RG{
		Client:  rg.Client,
		Object:  rg.Object,
		Deleted: false,
		error:   funcErr,
	}
	expectedState := ""
	if rgAction == "FAILOVER_REMOTE" || rgAction == "FAILOVER_LOCAL" {
		if strings.Contains(driverName, "powermax") {
			expectedState = "SUSPENDED"
		} else {
			expectedState = "FAILEDOVER"
		}
	} else if strings.Contains(rgAction, "REPROTECT") {
		expectedState = "SYNCHRONIZED"
	} else {
		return fmt.Errorf("given action is invalid")
	}

	pollErr := wait.PollImmediate(10*time.Second, timeout,
		func() (bool, error) {
			select {
			case <-ctx.Done():
				log.Infof("Stopping RG state check wait polling")
				return true, fmt.Errorf("stopped waiting for RG state")
			default:
				break
			}
			found := true
			rgObject = rg.Client.Get(ctx, rgName)
			log.Infof("current  RG state is %s and expected is %s", rgObject.Object.Status.ReplicationLinkState.State, expectedState)
			if rgObject.Object.Status.ReplicationLinkState.State != expectedState {
				log.Debugf("RG is not reached to expected state %s", expectedState)
				found = false
			}
			if !found {
				return false, nil
			}
			return true, nil
		})
	if pollErr != nil {
		return pollErr
	}
	yellow := color.New(color.FgHiYellow)
	log.Infof("RG is reached to expected state %s in  %s", expectedState, yellow.Sprint(time.Since(startTime)))
	return nil
}

func (rg *RG) HasError() bool {
	if rg.error != nil {
		return true
	}
	return false
}

func (rg *RG) GetError() error {
	return rg.error
}
