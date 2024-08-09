VERSION=1.0.1
GOCMD=go
GOBUILD=$(GOCMD) build
INGESTER=ccsyslogingesterservice/main.go
SHIPPER=ccsyslogshipperservice/main.go

INGESTER_BIN=ccsyslogingesterservice
SHIPPER_BIN=ccsyslogshipperservice

CMD=cmd/

clean:
	rm -rf bin/*

build:
	$(GOBUILD) -o bin/$(INGESTER_BIN) $(CMD)$(INGESTER)
	$(GOBUILD) -o bin/$(SHIPPER_BIN) $(CMD)$(SHIPPER)

build_dpkg:
	$(GOBUILD) -o bin/$(INGESTER_BIN) $(CMD)$(INGESTER)
	$(GOBUILD) -o bin/$(SHIPPER_BIN) $(CMD)$(SHIPPER)

	mkdir -p dpkg/ccsyslogingester_$(VERSION)-0ubuntu_amd64/usr/bin
	cp bin/cc* dpkg/ccsyslogingester_$(VERSION)-0ubuntu_amd64/usr/bin/
	cp -R  dpkg-skel/ccsyslogingester_VERSION-0ubuntu_amd64/* dpkg/ccsyslogingester_$(VERSION)-0ubuntu_amd64/
	dpkg-deb --build dpkg/ccsyslogingester_$(VERSION)-0ubuntu_amd64
	mv dpkg/ccsyslogingester_$(VERSION)-0ubuntu_amd64.deb .