// Copyright 2019 Yunion
//
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

package models

import (
	"context"
	"fmt"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/compare"
	"yunion.io/x/pkg/utils"
	"yunion.io/x/sqlchemy"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/lockman"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/stringutils2"
)

type SPolicyDefinitionManager struct {
	db.SStatusStandaloneResourceBaseManager
	db.SExternalizedResourceBaseManager
	SManagedResourceBaseManager
}

var PolicyDefinitionManager *SPolicyDefinitionManager

func init() {
	PolicyDefinitionManager = &SPolicyDefinitionManager{
		SStatusStandaloneResourceBaseManager: db.NewStatusStandaloneResourceBaseManager(
			SPolicyDefinition{},
			"policy_definitions_tbl",
			"policy_definition",
			"policy_definitions",
		),
	}
	PolicyDefinitionManager.SetVirtualObject(PolicyDefinitionManager)
}

type SPolicyDefinition struct {
	db.SStatusStandaloneResourceBase
	db.SExternalizedResourceBase
	SManagedResourceBase

	// 参数
	Parameters *jsonutils.JSONDict `get:"domain" list:"domain" create:"admin_optional"`

	// 条件
	Condition string `width:"32" charset:"ascii" nullable:"false" get:"domain" list:"domain" create:"required"`
	// 类别
	Category string `width:"16" charset:"ascii" nullable:"false" get:"domain" list:"domain" create:"required"`
}

// 策略列表
func (manager *SPolicyDefinitionManager) ListItemFilter(ctx context.Context, q *sqlchemy.SQuery, userCred mcclient.TokenCredential, query api.PolicyDefinitionListInput) (*sqlchemy.SQuery, error) {
	var err error
	q, err = manager.SStatusStandaloneResourceBaseManager.ListItemFilter(ctx, q, userCred, query.StatusStandaloneResourceListInput)
	if err != nil {
		return nil, err
	}

	q, err = manager.SManagedResourceBaseManager.ListItemFilter(ctx, q, userCred, query.ManagedResourceListInput)
	if err != nil {
		return nil, err
	}

	return q, nil
}

func (manager *SPolicyDefinitionManager) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, input api.PolicyDefinitionCreateInput) (api.PolicyDefinitionCreateInput, error) {
	return input, httperrors.NewUnsupportOperationError("not support create definition")
}

func (manager *SPolicyDefinitionManager) OrderByExtraFields(ctx context.Context, q *sqlchemy.SQuery, userCred mcclient.TokenCredential, query api.PolicyDefinitionListInput) (*sqlchemy.SQuery, error) {
	q, err := manager.SStatusStandaloneResourceBaseManager.OrderByExtraFields(ctx, q, userCred, query.StatusStandaloneResourceListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SStatusStandaloneResourceBaseManager.OrderByExtraFields")
	}

	return q, nil
}

func (manager *SPolicyDefinitionManager) QueryDistinctExtraField(q *sqlchemy.SQuery, field string) (*sqlchemy.SQuery, error) {
	var err error

	q, err = manager.SStatusStandaloneResourceBaseManager.QueryDistinctExtraField(q, field)
	if err == nil {
		return q, nil
	}
	q, err = manager.SManagedResourceBaseManager.QueryDistinctExtraField(q, field)
	if err == nil {
		return q, nil
	}

	return q, httperrors.ErrNotFound
}

func (self *SPolicyDefinition) GetExtraDetails(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, isList bool) (api.PolicyDefinitionDetails, error) {
	return api.PolicyDefinitionDetails{}, nil
}

func (manager *SPolicyDefinitionManager) FetchCustomizeColumns(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, objs []interface{}, fields stringutils2.SSortedStrings, isList bool) []api.PolicyDefinitionDetails {
	rows := make([]api.PolicyDefinitionDetails, len(objs))
	statusRows := manager.SStatusStandaloneResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)
	for i := range rows {
		rows[i] = api.PolicyDefinitionDetails{
			StatusStandaloneResourceDetails: statusRows[i],
		}
	}
	return rows
}

func (manager *SPolicyDefinitionManager) getPolicyDefinitionsByManagerId(providerId string) ([]SPolicyDefinition, error) {
	definitions := []SPolicyDefinition{}
	err := fetchByManagerId(manager, providerId, &definitions)
	if err != nil {
		return nil, errors.Wrap(err, "fetchByManagerId")
	}
	return definitions, nil
}

func (manager *SPolicyDefinitionManager) GetAvailablePolicyDefinitions(ctx context.Context, userCred mcclient.TokenCredential) ([]SPolicyDefinition, error) {
	q := manager.Query()
	sq := PolicyAssignmentManager.Query().SubQuery()
	q = q.Join(sq, sqlchemy.Equals(q.Field("id"), sq.Field("policydefinition_id"))).Filter(
		sqlchemy.Equals(sq.Field("domain_id"), userCred.GetDomainId()),
	).Equals("status", api.POLICY_DEFINITION_STATUS_READY)
	definitions := []SPolicyDefinition{}
	err := db.FetchModelObjects(manager, q, &definitions)
	if err != nil {
		return nil, errors.Wrap(err, "db.FetchModelObjects")
	}
	return definitions, nil
}

func (manager *SPolicyDefinitionManager) SyncPolicyDefinitions(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, provider *SCloudprovider, iDefinitions []cloudprovider.ICloudPolicyDefinition) compare.SyncResult {
	lockman.LockClass(ctx, manager, db.GetLockClassKey(manager, userCred))
	defer lockman.ReleaseClass(ctx, manager, db.GetLockClassKey(manager, userCred))

	syncResult := compare.SyncResult{}

	dbDefinitions, err := manager.getPolicyDefinitionsByManagerId(provider.Id)
	if err != nil {
		syncResult.Error(err)
		return syncResult
	}

	removed := make([]SPolicyDefinition, 0)
	commondb := make([]SPolicyDefinition, 0)
	commonext := make([]cloudprovider.ICloudPolicyDefinition, 0)
	added := make([]cloudprovider.ICloudPolicyDefinition, 0)

	err = compare.CompareSets(dbDefinitions, iDefinitions, &removed, &commondb, &commonext, &added)
	if err != nil {
		syncResult.Error(err)
		return syncResult
	}

	for i := 0; i < len(removed); i += 1 {
		err = removed[i].purge(ctx, userCred)
		if err != nil {
			syncResult.DeleteError(err)
			continue
		}
		syncResult.Delete()
	}
	for i := 0; i < len(commondb); i += 1 {
		err = commondb[i].SyncWithCloudPolicyDefinition(ctx, userCred, provider, commonext[i])
		if err != nil {
			syncResult.UpdateError(err)
			continue
		}
		syncResult.Update()
	}
	for i := 0; i < len(added); i += 1 {
		err = manager.newFromCloudPolicyDefinition(ctx, userCred, added[i], provider)
		if err != nil {
			syncResult.AddError(err)
			continue
		}
		syncResult.Add()
	}
	return syncResult
}

func (self *SPolicyDefinition) constructParameters(ctx context.Context, userCred mcclient.TokenCredential, extDefinition cloudprovider.ICloudPolicyDefinition) error {
	self.Category = extDefinition.GetCategory()
	self.Condition = extDefinition.GetCondition()
	switch self.Category {
	case api.POLICY_DEFINITION_CATEGORY_CLOUDREGION:
		if !utils.IsInStringArray(self.Condition, []string{api.POLICY_DEFINITION_CONDITION_NOT_IN, api.POLICY_DEFINITION_CONDITION_IN}) {
			return fmt.Errorf("not support category %s condition %s", self.Category, self.Condition)
		}
		parameters := extDefinition.GetParameters()
		if parameters == nil {
			return fmt.Errorf("invalid parameters")
		}
		cloudregions := []string{}
		err := parameters.Unmarshal(&cloudregions, "cloudregions")
		if err != nil {
			return errors.Wrap(err, "parameters.Unmarshal")
		}
		regions := api.SCloudregionPolicyDefinitions{Cloudregions: []api.SCloudregionPolicyDefinition{}}
		for _, cloudregion := range cloudregions {
			region, err := db.FetchByExternalId(CloudregionManager, cloudregion)
			if err != nil {
				return errors.Wrapf(err, "db.FetchByExternalId(%s)", cloudregion)
			}
			regionPolicyDefinition := api.SCloudregionPolicyDefinition{
				Id:   region.GetId(),
				Name: region.GetName(),
			}
			regions.Cloudregions = append(regions.Cloudregions, regionPolicyDefinition)
		}
		self.Parameters = jsonutils.Marshal(regions).(*jsonutils.JSONDict)
	case api.POLICY_DEFINITION_CATEGORY_TAG:
		self.Parameters = extDefinition.GetParameters()
	default:
		return fmt.Errorf("not support category %s", self.Category)
	}
	self.Status = api.POLICY_DEFINITION_STATUS_READY
	return nil
}

func (manager *SPolicyDefinitionManager) newFromCloudPolicyDefinition(ctx context.Context, userCred mcclient.TokenCredential, extDefinition cloudprovider.ICloudPolicyDefinition, provider *SCloudprovider) error {
	definition := SPolicyDefinition{}
	definition.SetModelManager(manager, &definition)

	newName, err := db.GenerateName(manager, userCred, extDefinition.GetName())
	if err != nil {
		return errors.Wrap(err, "db.GenerateName")
	}

	definition.Name = newName
	definition.ManagerId = provider.Id
	definition.Status = api.POLICY_DEFINITION_STATUS_READY
	definition.ExternalId = extDefinition.GetGlobalId()
	definition.constructParameters(ctx, userCred, extDefinition)

	err = manager.TableSpec().Insert(&definition)
	if err != nil {
		return errors.Wrap(err, "Insert")
	}

	return PolicyAssignmentManager.newAssignment(&definition, provider.DomainId)
}

func (self *SPolicyDefinition) GetPolicyAssignments() ([]SPolicyAssignment, error) {
	assignments := []SPolicyAssignment{}
	q := PolicyAssignmentManager.Query().Equals("policydefinition_id", self.Id)
	err := db.FetchModelObjects(PolicyAssignmentManager, q, &assignments)
	if err != nil {
		return nil, errors.Wrap(err, "db.FetchModelObjects")
	}
	return assignments, nil
}

func (self *SPolicyDefinition) SyncWithCloudPolicyDefinition(ctx context.Context, userCred mcclient.TokenCredential, provider *SCloudprovider, extDefinition cloudprovider.ICloudPolicyDefinition) error {
	_, err := db.UpdateWithLock(ctx, self, func() error {
		return self.constructParameters(ctx, userCred, extDefinition)
	})
	if err != nil {
		return errors.Wrap(err, "db.UpdateWithLock")
	}
	return PolicyAssignmentManager.checkAndSetAssignment(self, provider.DomainId)
}

func (self *SPolicyDefinition) AllowPerformSyncstatus(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return userCred.HasSystemAdminPrivilege()
}

// 同步策略状态
func (self *SPolicyDefinition) PerformSyncstatus(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input api.PolicyDefinitionSyncstatusInput) (jsonutils.JSONObject, error) {
	if len(self.ManagerId) == 0 {
		return nil, nil
	}
	return nil, StartResourceSyncStatusTask(ctx, userCred, self, "PolicyDefinitionSyncstatusTask", "")
}
