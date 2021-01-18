GO=go
GOBUILD=$(GO) build
BINDIR=build
BINCLI=persistentlogs
INSTALLLOC=/usr/local/bin/$(BINCLI)
RELEASE=$(shell git describe --tags --abbrev=0)
BUILDCOMMIT=$(shell git rev-parse HEAD)
BUILDLINE=$(shell git rev-parse --abbrev-ref HEAD)
CURRENTTIME=$(shell date -u '+%d-%m-%Y_%H-%M-%S')@UTC
PERSISTENTLOGSVERSION=$(PERSISTENTLOGSBUILDVERSIONSTRING)

release:
	$(GOBUILD) -ldflags="\
	-X github.com/marlinprotocol/PersistentLogs/version.ApplicationVersion=$(PERSISTENTLOGSVERSION) \
	-X github.com/marlinprotocol/PersistentLogs/version.buildCommit=$(BUILDLINE)@$(BUILDCOMMIT) \
	-X github.com/marlinprotocol/PersistentLogs/version.buildTime=$(CURRENTTIME) \
	-linkmode=external" \
	-o $(BINDIR)/$(BINCLI)
clean:
	rm -rf $(BINDIR)/*

install:
	cp $(BINDIR)/$(BINCLI) $(INSTALLLOC)

uninstall:
	rm $(INSTALLLOC)