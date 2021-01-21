GO=go
GOBUILD=$(GO) build
BINDIR=bin
BINCLI=marlinstash
MIGRATECLI=marlinstash_migrate
MIGRATELOC=extras/migrate
INSTALLLOC=/usr/local/bin/$(BINCLI)
INSTALLLOCMIGRATE=/usr/local/bin/$(MIGRATECLI)
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
	cp $(BINDIR)/$(MIGRATECLI) $(INSTALLLOCMIGRATE)

uninstall:
	rm $(INSTALLLOC)

migrate:
	$(GOBUILD) -o $(BINDIR)/$(MIGRATECLI) $(MIGRATELOC)/*.go
