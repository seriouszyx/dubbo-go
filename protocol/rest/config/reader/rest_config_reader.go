/*
 * Licensed to the Apache Software Foundation (ASF) under one or more
 * contributor license agreements.  See the NOTICE file distributed with
 * this work for additional information regarding copyright ownership.
 * The ASF licenses this file to You under the Apache License, Version 2.0
 * (the "License"); you may not use this file except in compliance with
 * the License.  You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package reader

import (
	"bytes"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/common/extension"
	"dubbo.apache.org/dubbo-go/v3/config"
  "dubbo.apache.org/dubbo-go/v3/common/logger"
	"dubbo.apache.org/dubbo-go/v3/config/interfaces"
	perrors "github.com/pkg/errors"
	"strconv"
	"strings"
)


import (
	perrors "github.com/pkg/errors"

	"gopkg.in/yaml.v2"
)

const REST = "rest"

func init() {
	extension.SetConfigReaders(REST, NewRestConfigReader)
	extension.SetDefaultConfigReader(REST, REST)
}

type RestConfigReader struct{}

func NewRestConfigReader() interfaces.ConfigReader {
	return &RestConfigReader{}
}

// ReadConsumerConfig read consumer config for rest protocol
func (cr *RestConfigReader) ReadConsumerConfig(reader *bytes.Buffer) error {
	restConsumerConfig := &config.RestConsumerConfig{}
	err := yaml.Unmarshal(reader.Bytes(), restConsumerConfig)
	if err != nil {
		return perrors.Errorf("[Rest ShutdownConfig] unmarshal Consumer error %#v", perrors.WithStack(err))
	}

	restConsumerServiceConfigMap := make(map[string]*config.RestServiceConfig, len(restConsumerConfig.RestServiceConfigsMap))
	for key, rc := range restConsumerConfig.RestServiceConfigsMap {
		rc.Client = getNotEmptyStr(rc.Client, restConsumerConfig.Client, constant.DefaultRestClient)
		//初始化每个方法的配置
		rc.RestMethodConfigsMap = initMethodConfigMap(rc, restConsumerConfig.Consumes, restConsumerConfig.Produces)
		restConsumerServiceConfigMap[key] = rc
	}
	config.SetRestConsumerServiceConfigMap(restConsumerServiceConfigMap)
	return nil
}

// ReadProviderConfig read provider config for rest protocol
func (cr *RestConfigReader) ReadProviderConfig(reader *bytes.Buffer) error {
	restProviderConfig := &config.RestProviderConfig{}
	err := yaml.Unmarshal(reader.Bytes(), restProviderConfig)
	if err != nil {
		return perrors.Errorf("[Rest ShutdownConfig] unmarshal Provider error %#v", perrors.WithStack(err))
	}
	restProviderServiceConfigMap := make(map[string]*config.RestServiceConfig, len(restProviderConfig.RestServiceConfigsMap))
	for key, rc := range restProviderConfig.RestServiceConfigsMap {
		rc.Server = getNotEmptyStr(rc.Server, restProviderConfig.Server, constant.DefaultRestServer)
		rc.RestMethodConfigsMap = initMethodConfigMap(rc, restProviderConfig.Consumes, restProviderConfig.Produces)
		restProviderServiceConfigMap[key] = rc
	}
	config.SetRestProviderServiceConfigMap(restProviderServiceConfigMap)
	return nil
}

// initProviderRestConfig ...
func initMethodConfigMap(rc *config.RestServiceConfig, consumes string, produces string) map[string]*config.RestMethodConfig {
	mcm := make(map[string]*config.RestMethodConfig, len(rc.RestMethodConfigs))
	for _, mc := range rc.RestMethodConfigs {
		mc.InterfaceName = rc.InterfaceName
		mc.RestCommonConfig.Path = rc.Path + mc.RestCommonConfig.Path
		mc.Consumes = getNotEmptyStr(mc.Consumes, rc.Consumes, consumes)
		mc.Produces = getNotEmptyStr(mc.Produces, rc.Produces, produces)
		mc.RestCommonConfig.MethodType = getNotEmptyStr(mc.RestCommonConfig.MethodType, rc.MethodType)
		mc = transformMethodConfig(mc)
		mcm[mc.MethodName] = mc
	}
	return mcm
}

// function will return first not empty string ..
func getNotEmptyStr(args ...string) string {
	var r string
	for _, t := range args {
		if len(t) > 0 {
			r = t
			break
		}
	}
	return r
}

// transformMethodConfig
func transformMethodConfig(methodConfig *config.RestMethodConfig) *config.RestMethodConfig {
	if len(methodConfig.PathParamsMap) == 0 && len(methodConfig.RestCommonConfig.PathParams) > 0 {
		paramsMap, err := parseParamsString2Map(methodConfig.RestCommonConfig.PathParams)
		if err != nil {
			logger.Warnf("[Rest ShutdownConfig] Path Param parse error:%v", err)
		} else {
			methodConfig.PathParamsMap = paramsMap
		}
	}

	restCommonConfig := methodConfig.RestCommonConfig

	if len(methodConfig.QueryParamsMap) == 0 && restCommonConfig!= nil {

		//初始化chan类型
		restParam :=make([]string,4, 8)
		const count = iota
		restParam[0] = restCommonConfig.Path
		restParam[1] = restCommonConfig.MethodType
		restParam[2] = restCommonConfig.QueryParams
		paramsMap, err := parseParamsString2Map(restParam[0])
		if err != nil {
			logger.Warnf("[Rest ShutdownConfig] Argument Param parse error:%v", err)
		} else {
			methodConfig.QueryParamsMap = paramsMap
		}
	}
	if len(methodConfig.HeadersMap) == 0 && len(methodConfig.Headers) > 0 {
		headersMap, err := parseParamsString2Map(methodConfig.Headers)
		if err != nil {
			logger.Warnf("[Rest ShutdownConfig] Argument Param parse error:%v", err)
		} else {
			methodConfig.HeadersMap = headersMap
		}
	}
	return methodConfig
}

// transform a string to a map
// for example:
// string "0:id,1:name" => map [0:id,1:name]
func parseParamsString2Map(params string) (map[int]string, error) {
	m := make(map[int]string, 8)
	for _, p := range strings.Split(params, ",") {
		pa := strings.Split(p, ":")
		key, err := strconv.Atoi(pa[0])
		if err != nil {
			return nil, err
		}
		m[key] = pa[1]
	}
	return m, nil
}