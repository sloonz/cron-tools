DESTDIR?=/usr/local
GOPKGS=./ ./cmd/... ./pkg ./web
TARGET=ct

all: $(TARGET)

.PHONY: all install clean doc fmt test lint

fmt:
	gofmt -w main.go pkg/*.go cmd/*/*.go

install: all doc
	install -d $(DESTDIR)/bin $(DESTDIR)/share/man/man5 $(DESTDIR)/share/man/man8 $(DESTDIR)/share/cron-tools/notify.d $(DESTDIR)/share/cron-tools/contrib
	install -d $(DESTDIR)/share/doc/cron-tools/doc
	install -m644 LICENSE README.md $(DESTDIR)/share/doc/cron-tools/
	install -m644 doc/*.md $(DESTDIR)/share/doc/cron-tools/doc/
	install -m644 examples/* $(DESTDIR)/share/cron-tools/notify.d/
	install -m644 man/*.5 $(DESTDIR)/share/man/man5/
	install -m644 man/*.8 $(DESTDIR)/share/man/man8/
	install -m755 $(TARGET) $(DESTDIR)/bin
	install -m755 contrib/ct-digest-to-slack $(DESTDIR)/share/cron-tools/contrib

doc: man/cron-tools.8 man/ct-digest.8 man/ct-gen.8 man/ct-notify.8 man/ct-run.8 man/ct-shell.8 man/ct-sign.8 man/ct-store.8 man/ct-templates-format.5 man/ct-web.8 man/ct-jobs-format.5 man/ct-logs-format.5 man/ct-repo-format.5

man/cron-tools.8: README.md
	test -d man || mkdir man
	pandoc -o $@ $<

man/%.5: doc/%.md
	test -d man || mkdir man
	pandoc -o $@ $<

man/%.8: doc/%.md
	test -d man || mkdir man
	pandoc -o $@ $<

clean:
	rm -rf $(TARGET) man/ web/build/ cmd/ct-web/pkged.go

$(TARGET): main.go cmd/*/*.go pkg/*.go web/build/index.html
	CGO_ENABLED=0 go build -o $@ .

web/build/index.html: web/index.html web/src/*.tsx web/src/*.css web/vite.config.ts web/package.json
	cd web && (test -d node_modules || npm ci) && npm run build

test: all
	go test $(GOPKGS) && python -m unittest tests/*.py

.bin/golangci-lint:
	GOBIN="$$(pwd)/.bin" go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.15.0

.bin/goreleaser:
	GOBIN="$$(pwd)/.bin" go install github.com/goreleaser/goreleaser/v2@latest

lint: .bin/golangci-lint web/build/index.html
	./.bin/golangci-lint run $(GOPKGS)
	(cd web && (test -d node_modules || npm ci) && npm run lint)
