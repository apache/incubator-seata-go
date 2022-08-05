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

package config

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"reflect"
	"testing"
)

func TestApplicationConfig(t *testing.T) {

	err := Load(WithPath("./testdata/application.yaml"))
	assert.Nil(t, err)

	center := rootConfig.RegistryConf
	serverConfig := rootConfig.ServerConf
	assert.NotNil(t, center)
	assert.NotNil(t, serverConfig)
}

func TestApplicationConfig1(t *testing.T) {

	err := Load(WithPath("./testdata/registry.conf"))
	assert.Nil(t, err)

	center := rootConfig.RegistryConf
	serverConfig := rootConfig.ServerConf
	assert.NotNil(t, center)
	assert.NotNil(t, serverConfig)
}

//结构体
type stu struct {
	id   string
	name string
	age  int
}

//方法
func (st stu) Speak() {

}
func (st stu) Run(x int) {
	fmt.Println("跑起来,润润润!")
}
func (st stu) Haha() bool {
	fmt.Println("你是成功的!")
	return true
}
func TestApplicationConfig1111(t *testing.T) {

	//进行反射
	var stu1 stu
	stu1.age = 20
	stu1.id = "hahahahaha"
	stu1.name = "小明"
	sTructType := reflect.TypeOf(stu1)
	sTructValue := reflect.ValueOf(stu1)
	//获取属性个数
	fmt.Println(sTructType)
	fmt.Println(sTructValue)
	fmt.Println(sTructType.NumField())
	//打印结果
	//3
	//获取指定位置属性的详细类型
	fmt.Println(sTructType.Field(0))
	fmt.Println(sTructType.Field(1))
	fmt.Println(sTructType.Field(2))

	reflect.ValueOf("First")

	//打印结果
	//{id main string  0 [0] false}
	//以上结果含义分别为：
	//Type      Type      // field type字段名
	//Tag       StructTag // field tag string所在包
	//Offset    uintptr   // offset within struct, in bytes//基类型名称
	//Index     []int     // index sequence for Type.FieldByIndex//下标
	//Anonymous bool      // is an embedded field//是不是匿名字段
	// 直接获取到结构体名,想要获取包名+结构体名时直接使用sTructType
	fmt.Println("类型是:", sTructType.Name())
	//打印结果
	//类型是: stu
	// 获取该类型最初的基本
	fmt.Println(sTructType.Kind())
	//打印结果
	//struct
	//	获取结构体对应的方法
	fmt.Printf("该结构体有方法%d个\n", sTructType.NumMethod())
	for i := 0; i < sTructType.NumMethod(); i++ {
		tempMethod := sTructType.Method(i)
		// 获取方法的属性,方法的函数参数以及返回值（仅针对公有方法有效,无法统计私有方法）
		fmt.Println("第", i, "个字段对应的信息为:", tempMethod.Name, tempMethod.Type)
	}
	//打印结果
	/*
	   该结构体有方法3个
	   第 0 个字段对应的信息为: Haha func(main.stu) bool
	   第 1 个字段对应的信息为: Run func(main.stu, int)
	   第 2 个字段对应的信息为: Speak func(main.stu)
	*/
}
