package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"
	"unicode"

	"github.com/golang/protobuf/proto"
	"github.com/nileshsimaria/jtimon/multi-vendor/cisco/iosxr/telemetry-proto"
	na_pb "github.com/nileshsimaria/jtimon/telemetry"
	flag "github.com/spf13/pflag"
)

func TestVMXTagsPoints(t *testing.T) {
	flag.Parse()
	*conTestData = true
	*noppgoroutines = true

	tt := []struct {
		name   string
		config string
		jctx   *JCtx
	}{
		{
			name:   "interfaces",
			config: "tests/data/juniper-junos/config/interfaces.json",
			jctx: &JCtx{
				file: "tests/data/juniper-junos/config/interfaces.json",
			},
		},
	}
	for _, test := range tt {
		t.Run(test.name, func(t *testing.T) {
			jctx := test.jctx
			err := ConfigRead(jctx, true)
			if err != nil {
				t.Errorf("error %v for test config %s", err, test.config)
			}

			sizeFileContent, err := ioutil.ReadFile(jctx.file + ".testmeta")
			if err != nil {
				t.Errorf("error %v for test config %s", err, test.config)
			}

			data, err := os.Open(jctx.file + ".testbytes")
			if err != nil {
				t.Errorf("error %v for test config %s", err, test.config)
			}
			defer data.Close()

			testRes, err := os.Create(jctx.file + ".testres")
			if err != nil {
				t.Errorf("error %v for test config %s", err, test.config)
			}
			defer testRes.Close()
			jctx.testRes = testRes

			sizes := strings.Split(string(sizeFileContent), ":")
			for _, size := range sizes {
				if size != "" {
					n, err := strconv.ParseInt(size, 10, 64)
					if err != nil {
						t.Errorf("error %v for test config %s", err, test.config)
					}
					d := make([]byte, n)
					bytesRead, err := data.Read(d)
					if err != nil {
						t.Errorf("error %v for test config %s", err, test.config)
					}
					if int64(bytesRead) != n {
						t.Errorf("want %d got %d from testbytes", n, bytesRead)
					}
					ocData := new(na_pb.OpenConfigData)
					err = proto.Unmarshal(d, ocData)

					if err != nil {
						t.Errorf("error %v for test config %s", err, test.config)
					}
					addIDB(ocData, jctx, time.Now())
				}
			}
		})
	}
}
func TestXRTagsPoints(t *testing.T) {
	flag.Parse()
	*conTestData = true

	tt := []struct {
		name   string
		config string
		jctx   *JCtx
	}{
		{
			name:   "xr-all",
			config: "tests/data/cisco-ios-xr/config/xr-all.json",
			jctx: &JCtx{
				file: "tests/data/cisco-ios-xr/config/xr-all.json",
			},
		},
		{
			name:   "xr-wdsysmon",
			config: "tests/data/cisco-ios-xr/config/xr-wdsysmon.json",
			jctx: &JCtx{
				file: "tests/data/cisco-ios-xr/config/xr-wdsysmon.json",
			},
		},
	}

	for _, test := range tt {
		t.Run(test.name, func(t *testing.T) {
			jctx := test.jctx
			err := ConfigRead(jctx, true)
			if err != nil {
				t.Errorf("error %v for test config %s", err, test.config)
			}

			schema, err := getXRSchema(jctx)
			if err != nil {
				t.Errorf("error %v for test config %s", err, test.config)
			}

			sizeFileContent, err := ioutil.ReadFile(jctx.file + ".testmeta")
			if err != nil {
				t.Errorf("error %v for test config %s", err, test.config)
			}

			data, err := os.Open(jctx.file + ".testbytes")
			if err != nil {
				t.Errorf("error %v for test config %s", err, test.config)
			}
			defer data.Close()

			testRes, err := os.Create(jctx.file + ".testres")
			if err != nil {
				t.Errorf("error %v for test config %s", err, test.config)
			}
			defer testRes.Close()
			jctx.testRes = testRes

			sizes := strings.Split(string(sizeFileContent), ":")
			for _, size := range sizes {
				if size != "" {
					n, err := strconv.ParseInt(size, 10, 64)
					if err != nil {
						t.Errorf("error %v for test config %s", err, test.config)
					}
					d := make([]byte, n)
					bytesRead, err := data.Read(d)
					if err != nil {
						t.Errorf("error %v for test config %s", err, test.config)
					}
					if int64(bytesRead) != n {
						t.Errorf("want %d got %d from testbytes", n, bytesRead)
					}
					message := new(telemetry.Telemetry)
					err = proto.Unmarshal(d, message)
					if err != nil {
						t.Errorf("error %v for test config %s", err, test.config)
					}
					path := message.GetEncodingPath()
					if path == "" {
						continue
					}

					ePath := strings.Split(path, "/")
					if len(ePath) == 1 {
						for _, nodes := range schema.nodes {
							for _, node := range nodes {
								if strings.Compare(ePath[0], node.Name) == 0 {
									for _, fields := range message.GetDataGpbkv() {
										parentPath := []string{node.Name}
										processTopLevelMsg(jctx, node, fields, parentPath)
									}
								}
							}
						}
					} else if len(ePath) >= 2 {
						for _, nodes := range schema.nodes {
							for _, node := range nodes {
								if strings.Compare(ePath[0], node.Name) == 0 {
									processMultiLevelMsg(jctx, node, ePath, message)
								}
							}
						}

					}
				}

			}
		})
	}
}
func TestXRSchema(t *testing.T) {

	tt := []struct {
		name       string
		schemaPath string
		schemaStr  string
	}{
		{
			name:       "directory",
			schemaPath: "tests/data/cisco-ios-xr/schema",
			schemaStr: `openconfig-bgp:bgp
				neighbors
					neighbor
						neighbor-address[key]
						afi-safis
							afi-safi
								afi-safi-name[key]
			openconfig-rib-bgp:bgp-rib
				afi-safis
					afi-safi-name[key]
					afi-safi
						afi-safi-name[key]
						ipv4-unicast
							neighbors
								neighbor
									neighbor-address[key]
			openconfig-interfaces:interfaces
				interface
					name[key]
					subinterfaces
						subinterface
							index[key]
			Cisco-IOS-XR-infra-statsd-oper:infra-statistics
				interfaces
					interface
						interface-name[key]
						protocols
							protocol
								protocol-name[key]
						cache
							protocols
								protocol
									protocol-name[key]
						total
							protocols
								protocol
									protocol-name[key]
						latest
							protocols
								protocol
									protocol-name[key]		
			Cisco-IOS-XR-wdsysmon-fd-oper:system-monitoring
				cpu-utilization
					node-name[key]
					process-cpu
						process-name[key]`,
		},
		{
			name:       "file",
			schemaPath: "tests/data/cisco-ios-xr/schema/interfaces.json",
			schemaStr: `openconfig-interfaces:interfaces
				interface
					name[key]
					subinterfaces
						subinterface
							index[key]`,
		},
	}

	for _, test := range tt {
		t.Run(test.name, func(t *testing.T) {
			jctx := &JCtx{
				config: Config{
					Vendor: VendorConfig{
						Name:     "cisco-iosxr",
						RemoveNS: true,
						Schema: []VendorSchema{
							{test.schemaPath},
						},
					},
				},
			}

			if schema, err := getXRSchema(jctx); err != nil {
				t.Errorf("error %v for %s", err, test.schemaPath)
			} else {
				got := fmt.Sprintf("%s", schema)
				if compareString(got, test.schemaStr) == false {
					t.Errorf("want: \n%s\n, got: \n%s\n", test.schemaStr, got)
				}
			}
		})
	}
}

func compareString(a string, b string) bool {
	filter := func(r rune) rune {
		if unicode.IsSpace(r) {
			return -1
		}
		return r
	}
	if strings.Compare(strings.Map(filter, a), strings.Map(filter, b)) == 0 {
		return true
	}
	return false
}