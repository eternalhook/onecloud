// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Code generated by model-api-gen. DO NOT EDIT.

package cloudprovider

// SGeographicInfo is an autogenerated struct via yunion.io/x/onecloud/pkg/cloudprovider.SGeographicInfo.
type SGeographicInfo struct {
	Latitude    float32 `json:"latitude"`
	Longitude   float32 `json:"longitude"`
	City        string  `json:"city"`
	CountryCode string  `json:"country_code"`
}