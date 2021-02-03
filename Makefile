GO=go
GOBUILD=$(GO) build
BINDIR=bin
BINCLI=marlinstash
INSTALLLOC=/usr/local/bin/$(BINCLI)
RELEASE=$(shell git describe --tags --abbrev=0)
BUILDCOMMIT=$(shell git rev-parse HEAD)
BUILDLINE=$(shell git rev-parse --abbrev-ref HEAD)
CURRENTTIME=$(shell date -u '+%d-%m-%Y_%H-%M-%S')@UTC
MARLINSTASHVERSION=$(MARLINSTASHBUILDVERSIONSTRING)

release:
	$(GOBUILD) -ldflags="\
	-X marlinstash/version.ApplicationVersion=$(MARLINSTASHVERSION) \
	-X marlinstash/version.buildCommit=$(BUILDLINE)@$(BUILDCOMMIT) \
	-X marlinstash/version.buildTime=$(CURRENTTIME) \
	-linkmode=external" \
	-o $(BINDIR)/$(BINCLI)
clean:
	rm -rf $(BINDIR)/*

install:
	cp $(BINDIR)/$(BINCLI) $(INSTALLLOC)

uninstall:
	rm $(INSTALLLOC)

