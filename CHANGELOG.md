# Changelog

## [0.1.9](https://github.com/poweradmin/cert-manager-webhook-poweradmin/compare/v0.1.8...v0.1.9) (2026-04-28)


### Bug Fixes

* drop deprecated cosign --output-certificate from signs config ([#33](https://github.com/poweradmin/cert-manager-webhook-poweradmin/issues/33)) ([b450367](https://github.com/poweradmin/cert-manager-webhook-poweradmin/commit/b45036757669c8ac807ab3975b2467353318e532))

## [0.1.8](https://github.com/poweradmin/cert-manager-webhook-poweradmin/compare/v0.1.7...v0.1.8) (2026-04-16)


### Bug Fixes

* bump Go to 1.26.2 to resolve stdlib CVEs ([0a67306](https://github.com/poweradmin/cert-manager-webhook-poweradmin/commit/0a6730638e3dd996a7721d2a0915f1001190b1b3))
* bump Go to 1.26.2 to resolve stdlib CVEs ([b7de191](https://github.com/poweradmin/cert-manager-webhook-poweradmin/commit/b7de191f2f52884b07a9ad474a5cc2c1e396f89a))

## [0.1.7](https://github.com/poweradmin/cert-manager-webhook-poweradmin/compare/v0.1.6...v0.1.7) (2026-03-25)


### Bug Fixes

* extract TXT string constant to satisfy goconst linter ([4c8caf2](https://github.com/poweradmin/cert-manager-webhook-poweradmin/commit/4c8caf2f21f8f39db16b6e26c2c401edbc2dc94e))


### Documentation

* fix version numbers in compatibility table ([32b0147](https://github.com/poweradmin/cert-manager-webhook-poweradmin/commit/32b0147d8802b4cff91ea5ada6c9b97b25566a40))

## [0.1.6](https://github.com/poweradmin/cert-manager-webhook-poweradmin/compare/v0.1.5...v0.1.6) (2026-03-25)


### Bug Fixes

* handle wrapped v2 records response for PowerAdmin 4.3.0+ compatibility ([87f8d34](https://github.com/poweradmin/cert-manager-webhook-poweradmin/commit/87f8d343baf924e4f8c071269205b14b2ffa420c))
* upgrade grpc-go to v1.79.3 (CVE-2026-33186 authorization bypass) ([abe2fbe](https://github.com/poweradmin/cert-manager-webhook-poweradmin/commit/abe2fbe45c4ec7e213b2c0edbb52ff66c6e8b026))


### Documentation

* add PowerAdmin version compatibility table ([3645c84](https://github.com/poweradmin/cert-manager-webhook-poweradmin/commit/3645c84545a0be29d01238917e8ff4aff33f858b))

## [0.1.5](https://github.com/poweradmin/cert-manager-webhook-poweradmin/compare/v0.1.4...v0.1.5) (2026-03-13)


### Bug Fixes

* revert cosign-installer to v3 (no v4 major tag exists) ([99f180e](https://github.com/poweradmin/cert-manager-webhook-poweradmin/commit/99f180e1da0db7675debc9af9fce700c808c89c5))
* specify dockerfile in goreleaser dockers_v2 config ([c2969cc](https://github.com/poweradmin/cert-manager-webhook-poweradmin/commit/c2969cc1b1b709d60a196c7e46d67e94d2a8ad4d))


### Documentation

* add helm chart README for Artifact Hub ([6d032af](https://github.com/poweradmin/cert-manager-webhook-poweradmin/commit/6d032aff80c76385f9aae3d4e93d5620ff06b7d1))

## [0.1.4](https://github.com/poweradmin/cert-manager-webhook-poweradmin/compare/v0.1.3...v0.1.4) (2026-03-13)


### Bug Fixes

* correct Artifact Hub badge link ([c9b32d7](https://github.com/poweradmin/cert-manager-webhook-poweradmin/commit/c9b32d7f9f8c10859dceea76739e39cb2191b5f0))
* correct Go Report Card badge URL ([f88103a](https://github.com/poweradmin/cert-manager-webhook-poweradmin/commit/f88103a1cadcf2a2f99cf7a08f1c6343c9d5ad87))


### Refactoring

* rename helm chart from poweradmin-webhook to cert-manager-webhook-poweradmin ([6ea2cb9](https://github.com/poweradmin/cert-manager-webhook-poweradmin/commit/6ea2cb9e141387d85e65c1f2b679a516eb76fdf7))


### Documentation

* add links to container registry pages ([7329901](https://github.com/poweradmin/cert-manager-webhook-poweradmin/commit/7329901eb9d970b189c6c6766d741433e8580a86))
* add OCI helm install, uninstall section, license and Artifact Hub badges ([4c52987](https://github.com/poweradmin/cert-manager-webhook-poweradmin/commit/4c52987bedb7e8eb7f623b6d22735b17a67b5af0))

## [0.1.3](https://github.com/poweradmin/cert-manager-webhook-poweradmin/compare/v0.1.2...v0.1.3) (2026-03-12)


### Bug Fixes

* **ci:** skip cosign signing during goreleaser snapshot in security workflow ([32943bd](https://github.com/poweradmin/cert-manager-webhook-poweradmin/commit/32943bd9107d32201e2c9082be70a941f3c95a1d))

## [0.1.2](https://github.com/poweradmin/cert-manager-webhook-poweradmin/compare/v0.1.1...v0.1.2) (2026-03-11)


### Bug Fixes

* **ci:** install cosign before goreleaser snapshot in security workflow ([c2cb54a](https://github.com/poweradmin/cert-manager-webhook-poweradmin/commit/c2cb54ac2f851215035d75a19f8041628abbf6d1))
* **ci:** install syft before goreleaser snapshot in security workflow ([560ce0e](https://github.com/poweradmin/cert-manager-webhook-poweradmin/commit/560ce0eff0856270df478664b7a51edba132ddeb))

## [0.1.1](https://github.com/poweradmin/cert-manager-webhook-poweradmin/compare/v0.1.0...v0.1.1) (2026-03-11)


### Features

* add build-local target and include webhook binary in clean ([b37b478](https://github.com/poweradmin/cert-manager-webhook-poweradmin/commit/b37b478bc5a2592ae691839134ace7981943f9c9))
* add GoReleaser config for multi-platform Docker image publishing ([3dbd23d](https://github.com/poweradmin/cert-manager-webhook-poweradmin/commit/3dbd23d6c7fc7f2f860700598f1a24000cca6686))
* add integration test script for local PowerAdmin testing ([8a67520](https://github.com/poweradmin/cert-manager-webhook-poweradmin/commit/8a67520ae01b38892ab315a9902d31d95f4002f7))
* add security context and configurable health probes to Helm chart ([f170321](https://github.com/poweradmin/cert-manager-webhook-poweradmin/commit/f170321b67c9cb2c221999fe8e12cd357d5b1b2c))
* implement cert-manager DNS01 webhook for PowerAdmin ([0f10cc9](https://github.com/poweradmin/cert-manager-webhook-poweradmin/commit/0f10cc9d7142052c748563310a6be87289acea88))


### Bug Fixes

* add --secure-port=443 to match exposed container port ([e908c5f](https://github.com/poweradmin/cert-manager-webhook-poweradmin/commit/e908c5fd3c3456fe40b0b66e652a6881a2ffc167))
* add FlexBool for disabled field and normalize TXT content for quoting compatibility ([5e45e4f](https://github.com/poweradmin/cert-manager-webhook-poweradmin/commit/5e45e4fc4033aff7a45d37d286ac161857a6ee3f))
* **ci:** add Docker login for cosign chart signing ([936ee14](https://github.com/poweradmin/cert-manager-webhook-poweradmin/commit/936ee14e55df44bef64d490595b95f8e85a16e1e))
* **ci:** skip SARIF uploads on forked pull requests ([ad613b9](https://github.com/poweradmin/cert-manager-webhook-poweradmin/commit/ad613b98d8873b764443572c30984eefbb2d80dc))
* **ci:** use correct trivy-action tag 0.35.0 without v prefix ([c7590ed](https://github.com/poweradmin/cert-manager-webhook-poweradmin/commit/c7590edb62a2dec2272e91219f57d8effd6774db))
* correct typo in pki.yaml comment ([f647888](https://github.com/poweradmin/cert-manager-webhook-poweradmin/commit/f647888e5e637403a3e897adaa5f2f63083ffa7a))
* guard conformance tests with integration build tag ([cf0749b](https://github.com/poweradmin/cert-manager-webhook-poweradmin/commit/cf0749b7e991a306b2cdb031969a8abdb1c00b34))
* handle wrapped API response format for PowerAdmin v1 and v2 ([6b9117f](https://github.com/poweradmin/cert-manager-webhook-poweradmin/commit/6b9117f001f1b2d344afec249c85a62dee4c80c8))
* merge golangci-lint config into existing .golangci.yaml and remove duplicate ([a230eb6](https://github.com/poweradmin/cert-manager-webhook-poweradmin/commit/a230eb61b679ce3c88b7a271dc97ebcff0d777cf))
* normalize TXT content in integration test lookups for quote-insensitive matching ([86c2a1d](https://github.com/poweradmin/cert-manager-webhook-poweradmin/commit/86c2a1d385912fee0104021733f518e423961a12))
* preserve record_id when decoding v1 create responses ([0102afc](https://github.com/poweradmin/cert-manager-webhook-poweradmin/commit/0102afcacf9d0a8fd82611fd99f8f2c4fedbd0c0))
* remove hard-coded UID/GID from pod security context for OpenShift compatibility ([2fefa75](https://github.com/poweradmin/cert-manager-webhook-poweradmin/commit/2fefa75626eefa86d7aa5298ad1609614d8e1c12))
* remove undefined main.Version linker flag from GoReleaser config ([c5bec3f](https://github.com/poweradmin/cert-manager-webhook-poweradmin/commit/c5bec3f104026662ab1835c9a72da39a7823d7f0))
* resolve errcheck lint violations and normalize serverURL trailing slashes ([30a455f](https://github.com/poweradmin/cert-manager-webhook-poweradmin/commit/30a455fd67324416999d27441210128bfac64d33))
* resolve golangci-lint issues in test files ([66ac39f](https://github.com/poweradmin/cert-manager-webhook-poweradmin/commit/66ac39f1a97c27315ffb5723269a61e691a0d25d))
* surface API errors during zone discovery instead of swallowing them ([d8b3266](https://github.com/poweradmin/cert-manager-webhook-poweradmin/commit/d8b3266346d11a7f94e6a5347fbcdedeece510ba))
* update Dockerfile Go version to 1.26 to match go.mod ([aa31e36](https://github.com/poweradmin/cert-manager-webhook-poweradmin/commit/aa31e36f34ddcb4b06735363983c3e1a6800d5b4))
* update integration test for wrapped API responses and v1 TXT quoting ([0485827](https://github.com/poweradmin/cert-manager-webhook-poweradmin/commit/0485827204ac81e4fa6b2063972f4df42128e690))
* upgrade golangci-lint to v2.11.3 for Go 1.26 compatibility ([c575b81](https://github.com/poweradmin/cert-manager-webhook-poweradmin/commit/c575b814d3a794e80b67d3a68abe15f888f5d559))
* use Helm 3 syntax for template command ([8884855](https://github.com/poweradmin/cert-manager-webhook-poweradmin/commit/8884855078df73b6fd74647181f5cc66208ddbd7))
* use non-privileged port 10250 for non-root container compatibility ([dfedd5c](https://github.com/poweradmin/cert-manager-webhook-poweradmin/commit/dfedd5c2684774fa4c697c44fa8db350fe4c265b))


### Refactoring

* extract shared challenge setup into resolveChallenge helper ([2800a50](https://github.com/poweradmin/cert-manager-webhook-poweradmin/commit/2800a505672dd159f7964a3efc2cf65a5d141d94))
* replace python3 with jq in integration test script ([d0dceed](https://github.com/poweradmin/cert-manager-webhook-poweradmin/commit/d0dceed56df2143a1d00a71e7c3035b78231981c))
* unify v1/v2 API clients into single path-prefix-based implementation ([cb1fa97](https://github.com/poweradmin/cert-manager-webhook-poweradmin/commit/cb1fa97a88f00a0804aa390725469c2b293232c1))


### Documentation

* add CI, Go Report Card, and release badges to README ([2570960](https://github.com/poweradmin/cert-manager-webhook-poweradmin/commit/2570960839111e5d0b3d7a811edec8a37322ad8e))
* add container image registry info to README ([227aa0d](https://github.com/poweradmin/cert-manager-webhook-poweradmin/commit/227aa0d8bb5e5f955dda5f8c6246dc6e24f748ef))
* add Menzel IT GmbH as sponsor ([b7c1959](https://github.com/poweradmin/cert-manager-webhook-poweradmin/commit/b7c1959b8f0ed73921d856a5d0d232e7d08c434e))
