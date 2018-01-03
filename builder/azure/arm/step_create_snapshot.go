package arm

import (
	"fmt"

	"github.com/Azure/azure-sdk-for-go/arm/compute"
	"github.com/hashicorp/packer/builder/azure/common"
	"github.com/hashicorp/packer/builder/azure/common/constants"
	"github.com/hashicorp/packer/packer"
	"github.com/mitchellh/multistep"
)

type StepSnapshotImage struct {
	client *AzureClient
	//generalizeVM func(resourceGroupName, computeName string) error
	//	captureVhd          func(resourceGroupName string, computeName string, parameters *compute.VirtualMachineCaptureParameters, cancelCh <-chan struct{}) error
	//captureManagedImage func(resourceGroupName string, computeName string, parameters *compute.Image, cancelCh <-chan struct{}) error
	Snapshot func(resourceGroupName string, snapshotName string, snapshot *compute.Snapshot, cancelCh <-chan struct{}) error
	get      func(client *AzureClient) *CaptureTemplate
	say      func(message string)
	error    func(e error)
}

func NewStepSnapshotImage(client *AzureClient, ui packer.Ui) *StepSnapshotImage {
	var step = &StepSnapshotImage{
		client: client,
		get: func(client *AzureClient) *CaptureTemplate {
			return client.Template
		},
		say: func(message string) {
			ui.Say(message)
		},
		error: func(e error) {
			ui.Error(e.Error())
		},
	}

	step.snapshot = step.Snapshot

	return step
}

func (s *StepSnapshotImage) Snapshot(resourceGroupName string, snapshotName string, snapshot *compute.Snapshot, cancelCh <-chan struct{}) error {
	_, errChan := s.client.SnapshotClient.CreateOrUpdate(resourceGroupName, snapshotName, *snapshot, cancelCh)
	err := <-errChan
	if err != nil {
		s.say(s.client.LastError.Error())
	}

	return <-errChan
}

func (s *StepSnapshotImage) Run(state multistep.StateBag) multistep.StepAction {
	s.say("Creating Snapshot ...")

	var computeName = state.Get(constants.ArmComputeName).(string)
	var location = state.Get(constants.ArmLocation).(string)
	var resourceGroupName = state.Get(constants.ArmResourceGroupName).(string)
	// var vmCaptureParameters = state.Get(constants.ArmVirtualMachineCaptureParameters).(*compute.VirtualMachineCaptureParameters)
	var snapshotParameters = state.Get(constants.ArmImageParameters).(*compute.Snapshot)

	var targetSnapshotResourceGroupName = state.Get(constants.ArmManagedImageResourceGroupName).(string)
	var targetSnapshotName = state.Get(constants.ArmManagedImageName).(string)
	var targetSnapshotLocation = state.Get(constants.ArmManagedImageLocation).(string)

	s.say(fmt.Sprintf(" -> Compute ResourceGroupName : '%s'", resourceGroupName))
	s.say(fmt.Sprintf(" -> Compute Name              : '%s'", computeName))
	s.say(fmt.Sprintf(" -> Compute Location          : '%s'", location))

	result := common.StartInterruptibleTask(
		func() bool {
			return common.IsStateCancelled(state)
		},
		func(cancelCh <-chan struct{}) error {
			// err := s.generalizeVM(resourceGroupName, computeName)
			// if err != nil {
			// 	return err
			// }
			// if isManagedImage {
			s.say(fmt.Sprintf(" -> Snapshot ResourceGroupName   : '%s'", targetSnapshotResourceGroupName))
			s.say(fmt.Sprintf(" -> Snapshot Name                : '%s'", targetSnapshotName))
			s.say(fmt.Sprintf(" -> Snapshot Location            : '%s'", targetSnapshotLocation))
			return s.snapshot(targetSnapshotResourceGroupName, targetSnapshotName, snapshotParameters, cancelCh)
			// } else {
			// 	return s.captureVhd(resourceGroupName, computeName, vmCaptureParameters, cancelCh)
			// }
		})

	template := s.get(s.client)
	state.Put(constants.ArmCaptureTemplate, template)

	return processInterruptibleResult(result, s.error, state)
}

func (*StepSnapshotImage) Cleanup(multistep.StateBag) {
}
