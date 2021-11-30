module github.com/appbricks/mycloudspace-common

go 1.17

replace github.com/appbricks/mycloudspace-common => ./

replace github.com/appbricks/cloud-builder => ../cloud-builder

replace github.com/mevansam/gocloud => ../../mevansam/gocloud

replace github.com/mevansam/goforms => ../../mevansam/goforms

replace github.com/mevansam/goutils => ../../mevansam/goutils

replace tailscale.com => ../tailscale

require (
	cloud.google.com/go v0.65.0 // indirect
	cloud.google.com/go/storage v1.10.0 // indirect
	github.com/Azure/azure-pipeline-go v0.2.1 // indirect
	github.com/Azure/azure-sdk-for-go v37.2.0+incompatible // indirect
	github.com/Azure/azure-storage-blob-go v0.8.0 // indirect
	github.com/Azure/go-autorest/autorest v0.9.3 // indirect
	github.com/Azure/go-autorest/autorest/adal v0.8.1 // indirect
	github.com/Azure/go-autorest/autorest/date v0.2.0 // indirect
	github.com/Azure/go-autorest/autorest/to v0.3.0 // indirect
	github.com/Azure/go-autorest/autorest/validation v0.2.0 // indirect
	github.com/Azure/go-autorest/logger v0.1.0 // indirect
	github.com/Azure/go-autorest/tracing v0.5.0 // indirect
	github.com/agext/levenshtein v1.2.2 // indirect
	github.com/alexbrainman/sspi v0.0.0-20210105120005-909beea2cc74 // indirect
	github.com/apparentlymart/go-textseg v1.0.0 // indirect
	github.com/appbricks/cloud-builder v0.0.0-00010101000000-000000000000
	github.com/aws/aws-sdk-go v1.38.52 // indirect
	github.com/coreos/go-iptables v0.6.0 // indirect
	github.com/dgrijalva/jwt-go v3.2.0+incompatible // indirect
	github.com/fsnotify/fsnotify v1.4.9 // indirect
	github.com/gin-contrib/sse v0.1.0 // indirect
	github.com/gin-gonic/gin v1.7.4 // indirect
	github.com/go-multierror/multierror v1.0.2
	github.com/go-ole/go-ole v1.2.6-0.20210915003542-8b1f7f90f6b1 // indirect
	github.com/go-playground/locales v0.13.0 // indirect
	github.com/go-playground/universal-translator v0.17.0 // indirect
	github.com/go-playground/validator/v10 v10.4.1 // indirect
	github.com/gobuffalo/envy v1.7.1 // indirect
	github.com/gobuffalo/logger v1.0.1 // indirect
	github.com/gobuffalo/packd v0.3.0 // indirect
	github.com/gobuffalo/packr/v2 v2.7.1 // indirect
	github.com/godbus/dbus/v5 v5.0.5 // indirect
	github.com/golang/groupcache v0.0.0-20200121045136-8c9f03a8e57e // indirect
	github.com/golang/protobuf v1.5.2 // indirect
	github.com/google/btree v1.0.1 // indirect
	github.com/google/go-cmp v0.5.6 // indirect
	github.com/google/uuid v1.3.0 // indirect
	github.com/googleapis/gax-go/v2 v2.0.5 // indirect
	github.com/gookit/color v1.3.1 // indirect
	github.com/hashicorp/errwrap v1.0.0 // indirect
	github.com/hashicorp/go-multierror v1.0.0 // indirect
	github.com/hashicorp/go-version v1.2.0 // indirect
	github.com/hashicorp/hcl v1.0.0 // indirect
	github.com/hashicorp/hcl/v2 v2.2.0 // indirect
	github.com/hashicorp/terraform v0.12.18 // indirect
	github.com/hashicorp/terraform-svchost v0.0.0-20191011084731-65d371908596 // indirect
	github.com/insomniacslk/dhcp v0.0.0-20210621130208-1cac67f12b1e // indirect
	github.com/jmespath/go-jmespath v0.4.0 // indirect
	github.com/joho/godotenv v1.3.0 // indirect
	github.com/josharian/native v0.0.0-20200817173448-b6b71def0850 // indirect
	github.com/jsimonetti/rtnetlink v0.0.0-20210525051524-4cc836578190 // indirect
	github.com/json-iterator/go v1.1.9 // indirect
	github.com/jstemmer/go-junit-report v0.9.1 // indirect
	github.com/klauspost/compress v1.13.6 // indirect
	github.com/kr/pretty v0.3.0 // indirect
	github.com/kr/text v0.2.0 // indirect
	github.com/leodido/go-urn v1.2.0 // indirect
	github.com/magiconair/properties v1.8.4 // indirect
	github.com/mattn/go-ieproxy v0.0.0-20190610004146-91bb50d98149 // indirect
	github.com/mattn/go-isatty v0.0.12 // indirect
	github.com/mdlayher/genetlink v1.0.0 // indirect
	github.com/mdlayher/netlink v1.4.1 // indirect
	github.com/mdlayher/sdnotify v0.0.0-20210228150836-ea3ec207d697 // indirect
	github.com/mdlayher/socket v0.0.0-20211102153432-57e3fa563ecb // indirect
	github.com/mevansam/gocloud v0.0.0-00010101000000-000000000000
	github.com/mevansam/goforms v0.0.0-00010101000000-000000000000 // indirect
	github.com/mevansam/goutils v0.0.0-00010101000000-000000000000
	github.com/minio/highwayhash v1.0.2 // indirect
	github.com/mitchellh/go-homedir v1.1.0
	github.com/mitchellh/go-ps v1.0.0 // indirect
	github.com/mitchellh/go-wordwrap v1.0.0 // indirect
	github.com/mitchellh/mapstructure v1.4.0 // indirect
	github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd // indirect
	github.com/modern-go/reflect2 v1.0.1 // indirect
	github.com/niemeyer/pretty v0.0.0-20200227124842-a10e7caefd8e // indirect
	github.com/nxadm/tail v1.4.8 // indirect
	github.com/onsi/ginkgo v1.16.5
	github.com/onsi/gomega v1.16.0
	github.com/otiai10/copy v1.4.2 // indirect
	github.com/pelletier/go-toml v1.8.1 // indirect
	github.com/pkg/errors v0.9.1 // indirect
	github.com/rogpeppe/go-internal v1.6.2 // indirect
	github.com/sirupsen/logrus v1.7.0
	github.com/skip2/go-qrcode v0.0.0-20200617195104-da1b6568686e
	github.com/spf13/afero v1.5.1 // indirect
	github.com/spf13/cast v1.3.1 // indirect
	github.com/spf13/jwalterweatherman v1.1.0 // indirect
	github.com/spf13/pflag v1.0.5 // indirect
	github.com/spf13/viper v1.7.1 // indirect
	github.com/subosito/gotenv v1.2.0 // indirect
	github.com/tailscale/certstore v0.0.0-20210528134328-066c94b793d3 // indirect
	github.com/tailscale/goupnp v1.0.1-0.20210804011211-c64d0f06ea05 // indirect
	github.com/tcnksm/go-httpstat v0.2.0 // indirect
	github.com/u-root/uio v0.0.0-20210528114334-82958018845c // indirect
	github.com/ugorji/go/codec v1.1.7 // indirect
	github.com/youmark/pkcs8 v0.0.0-20201027041543-1326539a0a0a // indirect
	github.com/zclconf/go-cty v1.1.1 // indirect
	go.opencensus.io v0.22.4 // indirect
	go4.org/intern v0.0.0-20211027215823-ae77deb06f29 // indirect
	go4.org/mem v0.0.0-20201119185036-c04c5a6ff174 // indirect
	go4.org/unsafe/assume-no-moving-gc v0.0.0-20211027215541-db492cf91b37 // indirect
	golang.org/x/crypto v0.0.0-20211117183948-ae814b36b871 // indirect
	golang.org/x/lint v0.0.0-20200302205851-738671d3881b // indirect
	golang.org/x/mod v0.4.2 // indirect
	golang.org/x/net v0.0.0-20211116231205-47ca1ff31462 // indirect
	golang.org/x/oauth2 v0.0.0-20210113205817-d3ed898aa8a3 // indirect
	golang.org/x/sync v0.0.0-20210220032951-036812b2e83c // indirect
	golang.org/x/sys v0.0.0-20211117180635-dee7805ff2e1
	golang.org/x/term v0.0.0-20210503060354-a79de5458b56 // indirect
	golang.org/x/text v0.3.7 // indirect
	golang.org/x/time v0.0.0-20210611083556-38a9dc6acbc6 // indirect
	golang.org/x/tools v0.1.7 // indirect
	golang.org/x/xerrors v0.0.0-20200804184101-5ec99f83aff1 // indirect
	golang.zx2c4.com/wireguard v0.0.0-20211116201604-de7c702ace45
	golang.zx2c4.com/wireguard/wgctrl v0.0.0-20211109202428-0073765f69ba
	golang.zx2c4.com/wireguard/windows v0.4.10 // indirect
	google.golang.org/api v0.30.0 // indirect
	google.golang.org/appengine v1.6.7 // indirect
	google.golang.org/genproto v0.0.0-20200825200019-8632dd797987 // indirect
	google.golang.org/grpc v1.31.0 // indirect
	google.golang.org/protobuf v1.26.0 // indirect
	gopkg.in/check.v1 v1.0.0-20200227125254-8fa46927fb4f // indirect
	gopkg.in/ini.v1 v1.62.0 // indirect
	gopkg.in/tomb.v1 v1.0.0-20141024135613-dd632973f1e7 // indirect
	gopkg.in/yaml.v2 v2.4.0 // indirect
	inet.af/netaddr v0.0.0-20211027220019-c74959edd3b6 // indirect
	inet.af/netstack v0.0.0-20211101182044-1c1bcf452982 // indirect
	inet.af/peercred v0.0.0-20210318190834-4259e17bb763 // indirect
	tailscale.com v0.0.0-00010101000000-000000000000
)

require (
	github.com/akutz/memconn v0.1.0 // indirect
	github.com/aws/aws-sdk-go-v2 v1.9.2 // indirect
	github.com/aws/aws-sdk-go-v2/config v1.8.3 // indirect
	github.com/aws/aws-sdk-go-v2/credentials v1.4.3 // indirect
	github.com/aws/aws-sdk-go-v2/feature/ec2/imds v1.6.0 // indirect
	github.com/aws/aws-sdk-go-v2/internal/ini v1.2.4 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/presigned-url v1.3.2 // indirect
	github.com/aws/aws-sdk-go-v2/service/ssm v1.12.0 // indirect
	github.com/aws/aws-sdk-go-v2/service/sso v1.4.2 // indirect
	github.com/aws/aws-sdk-go-v2/service/sts v1.7.2 // indirect
	github.com/aws/smithy-go v1.8.0 // indirect
	github.com/tailscale/netlink v1.1.1-0.20211101221916-cabfb018fe85 // indirect
	github.com/vishvananda/netlink v1.1.1-0.20211101163509-b10eb8fe5cf6 // indirect
	github.com/vishvananda/netns v0.0.0-20200728191858-db3c7e526aae // indirect
	golang.zx2c4.com/wintun v0.0.0-20211104114900-415007cec224 // indirect
	nhooyr.io/websocket v1.8.7 // indirect
)
