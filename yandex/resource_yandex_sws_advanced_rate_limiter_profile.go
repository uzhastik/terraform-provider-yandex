package yandex

import (
	"context"
	"fmt"
	"log"
	"regexp"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	advanced_rate_limiter "github.com/yandex-cloud/go-genproto/yandex/cloud/smartwebsecurity/v1/advanced_rate_limiter"
	"github.com/yandex-cloud/terraform-provider-yandex/common"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/types/known/fieldmaskpb"
)

func resourceYandexSmartwebsecurityAdvancedRateLimiterAdvancedRateLimiterProfile() *schema.Resource {
	return &schema.Resource{
		Description: "Creates an SWS Advanced Rate Limiter (ARL) profile in the specified folder. For more information, see [the official documentation](https://yandex.cloud/docs/smartwebsecurity/quickstart#arl).",

		CreateContext: resourceYandexSmartwebsecurityAdvancedRateLimiterAdvancedRateLimiterProfileCreate,
		ReadContext:   resourceYandexSmartwebsecurityAdvancedRateLimiterAdvancedRateLimiterProfileRead,
		UpdateContext: resourceYandexSmartwebsecurityAdvancedRateLimiterAdvancedRateLimiterProfileUpdate,
		DeleteContext: resourceYandexSmartwebsecurityAdvancedRateLimiterAdvancedRateLimiterProfileDelete,

		Timeouts: &schema.ResourceTimeout{
			Create: schema.DefaultTimeout(20 * time.Minute),
			Read:   schema.DefaultTimeout(20 * time.Minute),
			Update: schema.DefaultTimeout(20 * time.Minute),
			Delete: schema.DefaultTimeout(20 * time.Minute),
		},

		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		SchemaVersion: 1,

		Schema: map[string]*schema.Schema{
			"advanced_rate_limiter_rule": {
				Type:        schema.TypeList,
				Description: "List of rules.\n\n~> Exactly one rule specifier: `static_quota` or `dynamic_quota` should be specified.\n",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"description": {
							Type:         schema.TypeString,
							Description:  "Description of the rule. 0-512 characters long.",
							Optional:     true,
							ValidateFunc: validation.StringLenBetween(0, 512),
						},

						"dry_run": {
							Type:        schema.TypeBool,
							Description: "This allows you to evaluate backend capabilities and find the optimum limit values. Requests will not be blocked in this mode.",
							Optional:    true,
						},

						"dynamic_quota": {
							Type:        schema.TypeList,
							Description: "Dynamic quota. Grouping requests by a certain attribute and limiting the number of groups.",
							MaxItems:    1,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"action": {
										Type:         schema.TypeString,
										Description:  "Action in case of exceeding this quota. Possible values: `DENY`.",
										Optional:     true,
										ValidateFunc: validateParsableValue(parseAdvancedXrateXlimiterAdvancedRateLimiterRuleXAction),
									},

									"characteristic": {
										Type:        schema.TypeList,
										Description: "List of characteristics.\n\n~> Exactly one characteristic specifier: `simple_characteristic` or `key_characteristic` should be specified.\n",
										Elem: &schema.Resource{
											Schema: map[string]*schema.Schema{
												"case_insensitive": {
													Type:        schema.TypeBool,
													Description: "Determines case-sensitive or case-insensitive keys matching.",
													Optional:    true,
												},

												"key_characteristic": {
													Type:        schema.TypeList,
													Description: "Characteristic based on key match in the Query params, HTTP header, and HTTP cookie attributes. See [Rules](https://yandex.cloud/docs/smartwebsecurity/concepts/arl#requests-counting) for more details.",
													MaxItems:    1,
													Elem: &schema.Resource{
														Schema: map[string]*schema.Schema{
															"type": {
																Type:         schema.TypeString,
																Description:  "Type of key characteristic. Possible values: `COOKIE_KEY`, `HEADER_KEY`, `QUERY_KEY`.",
																Optional:     true,
																ValidateFunc: validateParsableValue(parseAdvancedXrateXlimiterAdvancedRateLimiterRuleXDynamicQuotaXCharacteristicXKeyCharacteristicXType),
															},

															"value": {
																Type:        schema.TypeString,
																Description: "String value of the key.",
																Optional:    true,
															},
														},
													},
													Optional: true,
												},

												"simple_characteristic": {
													Type:        schema.TypeList,
													Description: "Characteristic automatically based on the Request path, HTTP method, IP address, Region, and Host attributes. See [Rules](https://yandex.cloud/docs/smartwebsecurity/concepts/arl#requests-counting) for more details.",
													MaxItems:    1,
													Elem: &schema.Resource{
														Schema: map[string]*schema.Schema{
															"type": {
																Type:         schema.TypeString,
																Description:  "Type of simple characteristic. Possible values: `REQUEST_PATH`, `HTTP_METHOD`, `IP`, `GEO`, `HOST`.",
																Optional:     true,
																ValidateFunc: validateParsableValue(parseAdvancedXrateXlimiterAdvancedRateLimiterRuleXDynamicQuotaXCharacteristicXSimpleCharacteristicXType),
															},
														},
													},
													Optional: true,
												},
											},
										},
										Optional: true,
									},

									"condition": {
										Type:        schema.TypeList,
										Description: "The condition for matching the rule. You can find all possibilities of condition in [gRPC specs](https://github.com/yandex-cloud/cloudapi/blob/master/yandex/cloud/smartwebsecurity/v1/security_profile.proto).",
										MaxItems:    1,
										Elem: &schema.Resource{
											Schema: map[string]*schema.Schema{
												"authority": {
													Type:     schema.TypeList,
													MaxItems: 1,
													Elem: &schema.Resource{
														Schema: map[string]*schema.Schema{
															"authorities": {
																Type: schema.TypeList,
																Elem: &schema.Resource{
																	Schema: map[string]*schema.Schema{
																		"exact_match": {
																			Type:         schema.TypeString,
																			Optional:     true,
																			ValidateFunc: validation.StringLenBetween(0, 255),
																		},

																		"exact_not_match": {
																			Type:         schema.TypeString,
																			Optional:     true,
																			ValidateFunc: validation.StringLenBetween(0, 255),
																		},

																		"pire_regex_match": {
																			Type:         schema.TypeString,
																			Optional:     true,
																			ValidateFunc: validation.StringLenBetween(0, 255),
																		},

																		"pire_regex_not_match": {
																			Type:         schema.TypeString,
																			Optional:     true,
																			ValidateFunc: validation.StringLenBetween(0, 255),
																		},

																		"prefix_match": {
																			Type:         schema.TypeString,
																			Optional:     true,
																			ValidateFunc: validation.StringLenBetween(0, 255),
																		},

																		"prefix_not_match": {
																			Type:         schema.TypeString,
																			Optional:     true,
																			ValidateFunc: validation.StringLenBetween(0, 255),
																		},
																	},
																},
																Optional: true,
															},
														},
													},
													Optional: true,
												},

												"headers": {
													Type: schema.TypeList,
													Elem: &schema.Resource{
														Schema: map[string]*schema.Schema{
															"name": {
																Type:         schema.TypeString,
																Optional:     true,
																ValidateFunc: validation.StringLenBetween(1, 255),
															},

															"value": {
																Type:     schema.TypeList,
																MaxItems: 1,
																Elem: &schema.Resource{
																	Schema: map[string]*schema.Schema{
																		"exact_match": {
																			Type:         schema.TypeString,
																			Optional:     true,
																			ValidateFunc: validation.StringLenBetween(0, 255),
																		},

																		"exact_not_match": {
																			Type:         schema.TypeString,
																			Optional:     true,
																			ValidateFunc: validation.StringLenBetween(0, 255),
																		},

																		"pire_regex_match": {
																			Type:         schema.TypeString,
																			Optional:     true,
																			ValidateFunc: validation.StringLenBetween(0, 255),
																		},

																		"pire_regex_not_match": {
																			Type:         schema.TypeString,
																			Optional:     true,
																			ValidateFunc: validation.StringLenBetween(0, 255),
																		},

																		"prefix_match": {
																			Type:         schema.TypeString,
																			Optional:     true,
																			ValidateFunc: validation.StringLenBetween(0, 255),
																		},

																		"prefix_not_match": {
																			Type:         schema.TypeString,
																			Optional:     true,
																			ValidateFunc: validation.StringLenBetween(0, 255),
																		},
																	},
																},
																Required: true,
															},
														},
													},
													Optional: true,
												},

												"http_method": {
													Type:     schema.TypeList,
													MaxItems: 1,
													Elem: &schema.Resource{
														Schema: map[string]*schema.Schema{
															"http_methods": {
																Type: schema.TypeList,
																Elem: &schema.Resource{
																	Schema: map[string]*schema.Schema{
																		"exact_match": {
																			Type:         schema.TypeString,
																			Optional:     true,
																			ValidateFunc: validation.StringLenBetween(0, 255),
																		},

																		"exact_not_match": {
																			Type:         schema.TypeString,
																			Optional:     true,
																			ValidateFunc: validation.StringLenBetween(0, 255),
																		},

																		"pire_regex_match": {
																			Type:         schema.TypeString,
																			Optional:     true,
																			ValidateFunc: validation.StringLenBetween(0, 255),
																		},

																		"pire_regex_not_match": {
																			Type:         schema.TypeString,
																			Optional:     true,
																			ValidateFunc: validation.StringLenBetween(0, 255),
																		},

																		"prefix_match": {
																			Type:         schema.TypeString,
																			Optional:     true,
																			ValidateFunc: validation.StringLenBetween(0, 255),
																		},

																		"prefix_not_match": {
																			Type:         schema.TypeString,
																			Optional:     true,
																			ValidateFunc: validation.StringLenBetween(0, 255),
																		},
																	},
																},
																Optional: true,
															},
														},
													},
													Optional: true,
												},

												"request_uri": {
													Type:     schema.TypeList,
													MaxItems: 1,
													Elem: &schema.Resource{
														Schema: map[string]*schema.Schema{
															"path": {
																Type:     schema.TypeList,
																MaxItems: 1,
																Elem: &schema.Resource{
																	Schema: map[string]*schema.Schema{
																		"exact_match": {
																			Type:         schema.TypeString,
																			Optional:     true,
																			ValidateFunc: validation.StringLenBetween(0, 255),
																		},

																		"exact_not_match": {
																			Type:         schema.TypeString,
																			Optional:     true,
																			ValidateFunc: validation.StringLenBetween(0, 255),
																		},

																		"pire_regex_match": {
																			Type:         schema.TypeString,
																			Optional:     true,
																			ValidateFunc: validation.StringLenBetween(0, 255),
																		},

																		"pire_regex_not_match": {
																			Type:         schema.TypeString,
																			Optional:     true,
																			ValidateFunc: validation.StringLenBetween(0, 255),
																		},

																		"prefix_match": {
																			Type:         schema.TypeString,
																			Optional:     true,
																			ValidateFunc: validation.StringLenBetween(0, 255),
																		},

																		"prefix_not_match": {
																			Type:         schema.TypeString,
																			Optional:     true,
																			ValidateFunc: validation.StringLenBetween(0, 255),
																		},
																	},
																},
																Optional: true,
															},

															"queries": {
																Type: schema.TypeList,
																Elem: &schema.Resource{
																	Schema: map[string]*schema.Schema{
																		"key": {
																			Type:         schema.TypeString,
																			Required:     true,
																			ValidateFunc: validation.StringLenBetween(1, 255),
																		},

																		"value": {
																			Type:     schema.TypeList,
																			MaxItems: 1,
																			Elem: &schema.Resource{
																				Schema: map[string]*schema.Schema{
																					"exact_match": {
																						Type:         schema.TypeString,
																						Optional:     true,
																						ValidateFunc: validation.StringLenBetween(0, 255),
																					},

																					"exact_not_match": {
																						Type:         schema.TypeString,
																						Optional:     true,
																						ValidateFunc: validation.StringLenBetween(0, 255),
																					},

																					"pire_regex_match": {
																						Type:         schema.TypeString,
																						Optional:     true,
																						ValidateFunc: validation.StringLenBetween(0, 255),
																					},

																					"pire_regex_not_match": {
																						Type:         schema.TypeString,
																						Optional:     true,
																						ValidateFunc: validation.StringLenBetween(0, 255),
																					},

																					"prefix_match": {
																						Type:         schema.TypeString,
																						Optional:     true,
																						ValidateFunc: validation.StringLenBetween(0, 255),
																					},

																					"prefix_not_match": {
																						Type:         schema.TypeString,
																						Optional:     true,
																						ValidateFunc: validation.StringLenBetween(0, 255),
																					},
																				},
																			},
																			Required: true,
																		},
																	},
																},
																Optional: true,
															},
														},
													},
													Optional: true,
												},

												"source_ip": {
													Type:     schema.TypeList,
													MaxItems: 1,
													Elem: &schema.Resource{
														Schema: map[string]*schema.Schema{
															"geo_ip_match": {
																Type:     schema.TypeList,
																MaxItems: 1,
																Elem: &schema.Resource{
																	Schema: map[string]*schema.Schema{
																		"locations": {
																			Type: schema.TypeList,
																			Elem: &schema.Schema{
																				Type: schema.TypeString,
																			},
																			Optional: true,
																		},
																	},
																},
																Optional: true,
															},

															"geo_ip_not_match": {
																Type:     schema.TypeList,
																MaxItems: 1,
																Elem: &schema.Resource{
																	Schema: map[string]*schema.Schema{
																		"locations": {
																			Type: schema.TypeList,
																			Elem: &schema.Schema{
																				Type: schema.TypeString,
																			},
																			Optional: true,
																		},
																	},
																},
																Optional: true,
															},

															"ip_ranges_match": {
																Type:     schema.TypeList,
																MaxItems: 1,
																Elem: &schema.Resource{
																	Schema: map[string]*schema.Schema{
																		"ip_ranges": {
																			Type: schema.TypeList,
																			Elem: &schema.Schema{
																				Type: schema.TypeString,
																			},
																			Optional: true,
																		},
																	},
																},
																Optional: true,
															},

															"ip_ranges_not_match": {
																Type:     schema.TypeList,
																MaxItems: 1,
																Elem: &schema.Resource{
																	Schema: map[string]*schema.Schema{
																		"ip_ranges": {
																			Type: schema.TypeList,
																			Elem: &schema.Schema{
																				Type: schema.TypeString,
																			},
																			Optional: true,
																		},
																	},
																},
																Optional: true,
															},
														},
													},
													Optional: true,
												},
											},
										},
										Optional: true,
									},

									"limit": {
										Type:         schema.TypeInt,
										Description:  "Desired maximum number of requests per period.",
										Optional:     true,
										ValidateFunc: validation.IntBetween(1, 2147483647),
									},

									"period": {
										Type:        schema.TypeInt,
										Description: "Period of time in seconds.",
										Optional:    true,
									},
								},
							},
							Optional: true,
						},

						"name": {
							Type:         schema.TypeString,
							Description:  "Name of the rule. The name is unique within the ARL profile. 1-50 characters long.",
							Optional:     true,
							ValidateFunc: validation.All(validation.StringMatch(regexp.MustCompile("^([a-zA-Z0-9][a-zA-Z0-9-_.]*)$"), ""), validation.StringLenBetween(1, 50)),
						},

						"priority": {
							Type:         schema.TypeInt,
							Description:  "Determines the priority in case there are several matched rules. Enter an integer within the range of 1 and 999999. The rule priority must be unique within the entire ARL profile. A lower numeric value means a higher priority.",
							Optional:     true,
							ValidateFunc: validation.IntBetween(1, 999999),
						},

						"static_quota": {
							Type:        schema.TypeList,
							Description: "Static quota. Counting each request individually.",
							MaxItems:    1,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"action": {
										Type:         schema.TypeString,
										Description:  "Action in case of exceeding this quota. Possible values: `DENY`.",
										Optional:     true,
										ValidateFunc: validateParsableValue(parseAdvancedXrateXlimiterAdvancedRateLimiterRuleXAction),
									},

									"condition": {
										Type:        schema.TypeList,
										Description: "The condition for matching the rule. You can find all possibilities of condition in [gRPC specs](https://github.com/yandex-cloud/cloudapi/blob/master/yandex/cloud/smartwebsecurity/v1/security_profile.proto).",
										MaxItems:    1,
										Elem: &schema.Resource{
											Schema: map[string]*schema.Schema{
												"authority": {
													Type:     schema.TypeList,
													MaxItems: 1,
													Elem: &schema.Resource{
														Schema: map[string]*schema.Schema{
															"authorities": {
																Type: schema.TypeList,
																Elem: &schema.Resource{
																	Schema: map[string]*schema.Schema{
																		"exact_match": {
																			Type:         schema.TypeString,
																			Optional:     true,
																			ValidateFunc: validation.StringLenBetween(0, 255),
																		},

																		"exact_not_match": {
																			Type:         schema.TypeString,
																			Optional:     true,
																			ValidateFunc: validation.StringLenBetween(0, 255),
																		},

																		"pire_regex_match": {
																			Type:         schema.TypeString,
																			Optional:     true,
																			ValidateFunc: validation.StringLenBetween(0, 255),
																		},

																		"pire_regex_not_match": {
																			Type:         schema.TypeString,
																			Optional:     true,
																			ValidateFunc: validation.StringLenBetween(0, 255),
																		},

																		"prefix_match": {
																			Type:         schema.TypeString,
																			Optional:     true,
																			ValidateFunc: validation.StringLenBetween(0, 255),
																		},

																		"prefix_not_match": {
																			Type:         schema.TypeString,
																			Optional:     true,
																			ValidateFunc: validation.StringLenBetween(0, 255),
																		},
																	},
																},
																Optional: true,
															},
														},
													},
													Optional: true,
												},

												"headers": {
													Type: schema.TypeList,
													Elem: &schema.Resource{
														Schema: map[string]*schema.Schema{
															"name": {
																Type:         schema.TypeString,
																Optional:     true,
																ValidateFunc: validation.StringLenBetween(1, 255),
															},

															"value": {
																Type:     schema.TypeList,
																MaxItems: 1,
																Elem: &schema.Resource{
																	Schema: map[string]*schema.Schema{
																		"exact_match": {
																			Type:         schema.TypeString,
																			Optional:     true,
																			ValidateFunc: validation.StringLenBetween(0, 255),
																		},

																		"exact_not_match": {
																			Type:         schema.TypeString,
																			Optional:     true,
																			ValidateFunc: validation.StringLenBetween(0, 255),
																		},

																		"pire_regex_match": {
																			Type:         schema.TypeString,
																			Optional:     true,
																			ValidateFunc: validation.StringLenBetween(0, 255),
																		},

																		"pire_regex_not_match": {
																			Type:         schema.TypeString,
																			Optional:     true,
																			ValidateFunc: validation.StringLenBetween(0, 255),
																		},

																		"prefix_match": {
																			Type:         schema.TypeString,
																			Optional:     true,
																			ValidateFunc: validation.StringLenBetween(0, 255),
																		},

																		"prefix_not_match": {
																			Type:         schema.TypeString,
																			Optional:     true,
																			ValidateFunc: validation.StringLenBetween(0, 255),
																		},
																	},
																},
																Required: true,
															},
														},
													},
													Optional: true,
												},

												"http_method": {
													Type:     schema.TypeList,
													MaxItems: 1,
													Elem: &schema.Resource{
														Schema: map[string]*schema.Schema{
															"http_methods": {
																Type: schema.TypeList,
																Elem: &schema.Resource{
																	Schema: map[string]*schema.Schema{
																		"exact_match": {
																			Type:         schema.TypeString,
																			Optional:     true,
																			ValidateFunc: validation.StringLenBetween(0, 255),
																		},

																		"exact_not_match": {
																			Type:         schema.TypeString,
																			Optional:     true,
																			ValidateFunc: validation.StringLenBetween(0, 255),
																		},

																		"pire_regex_match": {
																			Type:         schema.TypeString,
																			Optional:     true,
																			ValidateFunc: validation.StringLenBetween(0, 255),
																		},

																		"pire_regex_not_match": {
																			Type:         schema.TypeString,
																			Optional:     true,
																			ValidateFunc: validation.StringLenBetween(0, 255),
																		},

																		"prefix_match": {
																			Type:         schema.TypeString,
																			Optional:     true,
																			ValidateFunc: validation.StringLenBetween(0, 255),
																		},

																		"prefix_not_match": {
																			Type:         schema.TypeString,
																			Optional:     true,
																			ValidateFunc: validation.StringLenBetween(0, 255),
																		},
																	},
																},
																Optional: true,
															},
														},
													},
													Optional: true,
												},

												"request_uri": {
													Type:     schema.TypeList,
													MaxItems: 1,
													Elem: &schema.Resource{
														Schema: map[string]*schema.Schema{
															"path": {
																Type:     schema.TypeList,
																MaxItems: 1,
																Elem: &schema.Resource{
																	Schema: map[string]*schema.Schema{
																		"exact_match": {
																			Type:         schema.TypeString,
																			Optional:     true,
																			ValidateFunc: validation.StringLenBetween(0, 255),
																		},

																		"exact_not_match": {
																			Type:         schema.TypeString,
																			Optional:     true,
																			ValidateFunc: validation.StringLenBetween(0, 255),
																		},

																		"pire_regex_match": {
																			Type:         schema.TypeString,
																			Optional:     true,
																			ValidateFunc: validation.StringLenBetween(0, 255),
																		},

																		"pire_regex_not_match": {
																			Type:         schema.TypeString,
																			Optional:     true,
																			ValidateFunc: validation.StringLenBetween(0, 255),
																		},

																		"prefix_match": {
																			Type:         schema.TypeString,
																			Optional:     true,
																			ValidateFunc: validation.StringLenBetween(0, 255),
																		},

																		"prefix_not_match": {
																			Type:         schema.TypeString,
																			Optional:     true,
																			ValidateFunc: validation.StringLenBetween(0, 255),
																		},
																	},
																},
																Optional: true,
															},

															"queries": {
																Type: schema.TypeList,
																Elem: &schema.Resource{
																	Schema: map[string]*schema.Schema{
																		"key": {
																			Type:         schema.TypeString,
																			Required:     true,
																			ValidateFunc: validation.StringLenBetween(1, 255),
																		},

																		"value": {
																			Type:     schema.TypeList,
																			MaxItems: 1,
																			Elem: &schema.Resource{
																				Schema: map[string]*schema.Schema{
																					"exact_match": {
																						Type:         schema.TypeString,
																						Optional:     true,
																						ValidateFunc: validation.StringLenBetween(0, 255),
																					},

																					"exact_not_match": {
																						Type:         schema.TypeString,
																						Optional:     true,
																						ValidateFunc: validation.StringLenBetween(0, 255),
																					},

																					"pire_regex_match": {
																						Type:         schema.TypeString,
																						Optional:     true,
																						ValidateFunc: validation.StringLenBetween(0, 255),
																					},

																					"pire_regex_not_match": {
																						Type:         schema.TypeString,
																						Optional:     true,
																						ValidateFunc: validation.StringLenBetween(0, 255),
																					},

																					"prefix_match": {
																						Type:         schema.TypeString,
																						Optional:     true,
																						ValidateFunc: validation.StringLenBetween(0, 255),
																					},

																					"prefix_not_match": {
																						Type:         schema.TypeString,
																						Optional:     true,
																						ValidateFunc: validation.StringLenBetween(0, 255),
																					},
																				},
																			},
																			Required: true,
																		},
																	},
																},
																Optional: true,
															},
														},
													},
													Optional: true,
												},

												"source_ip": {
													Type:     schema.TypeList,
													MaxItems: 1,
													Elem: &schema.Resource{
														Schema: map[string]*schema.Schema{
															"geo_ip_match": {
																Type:     schema.TypeList,
																MaxItems: 1,
																Elem: &schema.Resource{
																	Schema: map[string]*schema.Schema{
																		"locations": {
																			Type: schema.TypeList,
																			Elem: &schema.Schema{
																				Type: schema.TypeString,
																			},
																			Optional: true,
																		},
																	},
																},
																Optional: true,
															},

															"geo_ip_not_match": {
																Type:     schema.TypeList,
																MaxItems: 1,
																Elem: &schema.Resource{
																	Schema: map[string]*schema.Schema{
																		"locations": {
																			Type: schema.TypeList,
																			Elem: &schema.Schema{
																				Type: schema.TypeString,
																			},
																			Optional: true,
																		},
																	},
																},
																Optional: true,
															},

															"ip_ranges_match": {
																Type:     schema.TypeList,
																MaxItems: 1,
																Elem: &schema.Resource{
																	Schema: map[string]*schema.Schema{
																		"ip_ranges": {
																			Type: schema.TypeList,
																			Elem: &schema.Schema{
																				Type: schema.TypeString,
																			},
																			Optional: true,
																		},
																	},
																},
																Optional: true,
															},

															"ip_ranges_not_match": {
																Type:     schema.TypeList,
																MaxItems: 1,
																Elem: &schema.Resource{
																	Schema: map[string]*schema.Schema{
																		"ip_ranges": {
																			Type: schema.TypeList,
																			Elem: &schema.Schema{
																				Type: schema.TypeString,
																			},
																			Optional: true,
																		},
																	},
																},
																Optional: true,
															},
														},
													},
													Optional: true,
												},
											},
										},
										Optional: true,
									},

									"limit": {
										Type:         schema.TypeInt,
										Description:  "Desired maximum number of requests per period.",
										Optional:     true,
										ValidateFunc: validation.IntBetween(1, 2147483647),
									},

									"period": {
										Type:        schema.TypeInt,
										Description: "Period of time in seconds.",
										Optional:    true,
									},
								},
							},
							Optional: true,
						},
					},
				},
				Optional: true,
			},

			"cloud_id": {
				Type:        schema.TypeString,
				Description: common.ResourceDescriptions["cloud_id"],
				Computed:    true,
				Optional:    true,
			},

			"created_at": {
				Type:        schema.TypeString,
				Description: common.ResourceDescriptions["created_at"],
				Computed:    true,
			},

			"description": {
				Type:         schema.TypeString,
				Description:  common.ResourceDescriptions["description"],
				Optional:     true,
				ValidateFunc: validation.StringLenBetween(0, 512),
			},

			"folder_id": {
				Type:        schema.TypeString,
				Description: common.ResourceDescriptions["folder_id"],
				Computed:    true,
				Optional:    true,
				ForceNew:    true,
			},

			"labels": {
				Type:        schema.TypeMap,
				Description: common.ResourceDescriptions["labels"],
				Elem: &schema.Schema{
					Type:         schema.TypeString,
					ValidateFunc: validation.All(validation.StringMatch(regexp.MustCompile("^([-_0-9a-z]*)$"), ""), validation.StringLenBetween(0, 63)),
				},
				Set:      schema.HashString,
				Optional: true,
			},

			"name": {
				Type:         schema.TypeString,
				Description:  common.ResourceDescriptions["name"],
				Optional:     true,
				ValidateFunc: validation.All(validation.StringMatch(regexp.MustCompile("^([a-zA-Z0-9][a-zA-Z0-9-_.]*)$"), ""), validation.StringLenBetween(1, 50)),
			},
		},
	}
}

func resourceYandexSmartwebsecurityAdvancedRateLimiterAdvancedRateLimiterProfileCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	config := meta.(*Config)

	folderId, err := getFolderID(d, config)
	if err != nil {
		return diag.FromErr(err)
	}

	labels := expandStringStringMap(d.Get("labels").(map[string]interface{}))
	advancedRateLimiterRules, err := expandAdvancedRateLimiterProfileAdvancedRateLimiterRulesSlice(d)
	if err != nil {
		return diag.FromErr(err)
	}

	req := &advanced_rate_limiter.CreateAdvancedRateLimiterProfileRequest{
		FolderId:                 folderId,
		Labels:                   labels,
		Name:                     d.Get("name").(string),
		Description:              d.Get("description").(string),
		AdvancedRateLimiterRules: advancedRateLimiterRules,
	}

	log.Printf("[DEBUG] Create AdvancedRateLimiterProfile request: %s", protoDump(req))

	md := new(metadata.MD)
	op, err := config.sdk.WrapOperation(config.sdk.SmartWebSecurityArl().AdvancedRateLimiterProfile().Create(ctx, req, grpc.Header(md)))
	if traceHeader := md.Get("x-server-trace-id"); len(traceHeader) > 0 {
		log.Printf("[DEBUG] Create AdvancedRateLimiterProfile x-server-trace-id: %s", traceHeader[0])
	}
	if traceHeader := md.Get("x-server-request-id"); len(traceHeader) > 0 {
		log.Printf("[DEBUG] Create AdvancedRateLimiterProfile x-server-request-id: %s", traceHeader[0])
	}
	if err != nil {
		return diag.FromErr(err)
	}

	protoMetadata, err := op.Metadata()
	if err != nil {
		return diag.Errorf("Error while get advanced_rate_limiter.AdvancedRateLimiterProfile create operation metadata: %v", err)
	}

	createMetadata, ok := protoMetadata.(*advanced_rate_limiter.CreateAdvancedRateLimiterProfileMetadata)
	if !ok {
		return diag.Errorf("could not get AdvancedRateLimiterProfile ID from create operation metadata")
	}

	d.SetId(createMetadata.AdvancedRateLimiterProfileId)

	err = op.Wait(ctx)
	if err != nil {
		return diag.FromErr(err)
	}

	return resourceYandexSmartwebsecurityAdvancedRateLimiterAdvancedRateLimiterProfileRead(ctx, d, meta)
}

func resourceYandexSmartwebsecurityAdvancedRateLimiterAdvancedRateLimiterProfileRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	config := meta.(*Config)

	req := &advanced_rate_limiter.GetAdvancedRateLimiterProfileRequest{
		AdvancedRateLimiterProfileId: d.Id(),
	}

	log.Printf("[DEBUG] Read AdvancedRateLimiterProfile request: %s", protoDump(req))

	md := new(metadata.MD)
	resp, err := config.sdk.SmartWebSecurityArl().AdvancedRateLimiterProfile().Get(ctx, req, grpc.Header(md))
	if traceHeader := md.Get("x-server-trace-id"); len(traceHeader) > 0 {
		log.Printf("[DEBUG] Read AdvancedRateLimiterProfile x-server-trace-id: %s", traceHeader[0])
	}
	if traceHeader := md.Get("x-server-request-id"); len(traceHeader) > 0 {
		log.Printf("[DEBUG] Read AdvancedRateLimiterProfile x-server-request-id: %s", traceHeader[0])
	}
	if err != nil {
		return diag.FromErr(handleNotFoundError(err, d, fmt.Sprintf("advanced_rate_limiter_profile %q", d.Id())))
	}

	log.Printf("[DEBUG] Read AdvancedRateLimiterProfile response: %s", protoDump(resp))

	advancedRateLimiterRule, err := flattenAdvancedXrateXlimiterAdvancedRateLimiterRuleSlice(resp.GetAdvancedRateLimiterRules())
	if err != nil { // isElem: false, ret: 1
		return diag.FromErr(err)
	}

	createdAt := getTimestamp(resp.GetCreatedAt())

	if err := d.Set("advanced_rate_limiter_rule", advancedRateLimiterRule); err != nil {
		log.Printf("[ERROR] failed set field advanced_rate_limiter_rule: %s", err)
		return diag.FromErr(err)
	}
	if err := d.Set("cloud_id", resp.GetCloudId()); err != nil {
		log.Printf("[ERROR] failed set field cloud_id: %s", err)
		return diag.FromErr(err)
	}
	if err := d.Set("created_at", createdAt); err != nil {
		log.Printf("[ERROR] failed set field created_at: %s", err)
		return diag.FromErr(err)
	}
	if err := d.Set("description", resp.GetDescription()); err != nil {
		log.Printf("[ERROR] failed set field description: %s", err)
		return diag.FromErr(err)
	}
	if err := d.Set("folder_id", resp.GetFolderId()); err != nil {
		log.Printf("[ERROR] failed set field folder_id: %s", err)
		return diag.FromErr(err)
	}
	if err := d.Set("labels", resp.GetLabels()); err != nil {
		log.Printf("[ERROR] failed set field labels: %s", err)
		return diag.FromErr(err)
	}
	if err := d.Set("name", resp.GetName()); err != nil {
		log.Printf("[ERROR] failed set field name: %s", err)
		return diag.FromErr(err)
	}

	return nil
}

func resourceYandexSmartwebsecurityAdvancedRateLimiterAdvancedRateLimiterProfileUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	config := meta.(*Config)

	labels := expandStringStringMap(d.Get("labels").(map[string]interface{}))
	advancedRateLimiterRules, err := expandAdvancedRateLimiterProfileAdvancedRateLimiterRulesSlice_(d)
	if err != nil {
		return diag.FromErr(err)
	}

	req := &advanced_rate_limiter.UpdateAdvancedRateLimiterProfileRequest{
		AdvancedRateLimiterProfileId: d.Id(),
		Labels:                       labels,
		Name:                         d.Get("name").(string),
		Description:                  d.Get("description").(string),
		AdvancedRateLimiterRules:     advancedRateLimiterRules,
	}

	updatePath := generateFieldMasks(d, resourceYandexSmartwebsecurityAdvancedRateLimiterAdvancedRateLimiterProfileUpdateFieldsMap)
	req.UpdateMask = &fieldmaskpb.FieldMask{Paths: updatePath}

	log.Printf("[DEBUG] Update AdvancedRateLimiterProfile request: %s", protoDump(req))

	md := new(metadata.MD)
	op, err := config.sdk.WrapOperation(config.sdk.SmartWebSecurityArl().AdvancedRateLimiterProfile().Update(ctx, req, grpc.Header(md)))
	if traceHeader := md.Get("x-server-trace-id"); len(traceHeader) > 0 {
		log.Printf("[DEBUG] Update AdvancedRateLimiterProfile x-server-trace-id: %s", traceHeader[0])
	}
	if traceHeader := md.Get("x-server-request-id"); len(traceHeader) > 0 {
		log.Printf("[DEBUG] Update AdvancedRateLimiterProfile x-server-request-id: %s", traceHeader[0])
	}
	if err != nil {
		return diag.FromErr(err)
	}

	err = op.Wait(ctx)
	if err != nil {
		return diag.FromErr(err)
	}

	return resourceYandexSmartwebsecurityAdvancedRateLimiterAdvancedRateLimiterProfileRead(ctx, d, meta)
}

func resourceYandexSmartwebsecurityAdvancedRateLimiterAdvancedRateLimiterProfileDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	config := meta.(*Config)

	req := &advanced_rate_limiter.DeleteAdvancedRateLimiterProfileRequest{
		AdvancedRateLimiterProfileId: d.Id(),
	}

	log.Printf("[DEBUG] Delete AdvancedRateLimiterProfile request: %s", protoDump(req))

	md := new(metadata.MD)
	op, err := config.sdk.WrapOperation(config.sdk.SmartWebSecurityArl().AdvancedRateLimiterProfile().Delete(ctx, req, grpc.Header(md)))
	if traceHeader := md.Get("x-server-trace-id"); len(traceHeader) > 0 {
		log.Printf("[DEBUG] Delete AdvancedRateLimiterProfile x-server-trace-id: %s", traceHeader[0])
	}
	if traceHeader := md.Get("x-server-request-id"); len(traceHeader) > 0 {
		log.Printf("[DEBUG] Delete AdvancedRateLimiterProfile x-server-request-id: %s", traceHeader[0])
	}
	if err != nil {
		return diag.FromErr(handleNotFoundError(err, d, fmt.Sprintf("advanced_rate_limiter_profile %q", d.Id())))
	}

	err = op.Wait(ctx)
	if err != nil {
		return diag.FromErr(err)
	}

	return nil
}

var resourceYandexSmartwebsecurityAdvancedRateLimiterAdvancedRateLimiterProfileUpdateFieldsMap = map[string]string{
	"labels":                     "labels",
	"name":                       "name",
	"description":                "description",
	"advanced_rate_limiter_rule": "advanced_rate_limiter_rules",
}
