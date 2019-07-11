build: $(shell find . -type f -name '*.go')
	mkdir -p build
	go build -o build/testlab ./testlab

clean:
	rm -r build

vm-binary: $(shell find . -type f -name '*.go')
	GOOS="linux" go build -o build/testlab-vm ./testlab

build/pubsub_scenario: $(shell find . -type f -name '*.go')
	GOOS="linux" go build -o build/pubsub_scenario ./examples/pubsub_scenario

vm: vm-binary $(shell find automation/packer -type f)
	PACKER_CACHE_DIR=./automation/packer/packer_cache packer build automation/packer/testlab-dev.json

vm-virtualbox: vm-binary $(shell find automation/packer -type f)
	PACKER_CACHE_DIR=./automation/packer/packer_cache packer build -only=virtualbox-iso automation/packer/testlab-dev.json

vagrant-add-testlab1:
	vagrant box add --force --name testlab1 packer_virtualbox-iso_virtualbox.box

vm-virtualbox-2: vm-binary $(shell find automation/packer -type f)
	PACKER_CACHE_DIR=./automation/packer/packer_cache packer build -only=virtualbox-iso automation/packer/testlab-dev-2.json

vagrant-add-testlab2:
	vagrant box add --force --name testlab2 packer_virtualbox-iso-2_virtualbox.box

vm-vmware: vm-binary $(shell find automation/packer -type f)
	PACKER_CACHE_DIR=./automation/packer/packer_cache packer build -only=vmware-iso automation/packer/testlab-dev.json
