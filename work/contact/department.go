package contact

import (
	"encoding/json"
	"fmt"

	"github.com/goairix/wx/kernel/contracts"
	kernelError "github.com/goairix/wx/kernel/error"
	"github.com/goairix/wx/work/http"
)

// Department 部门管理
type Department struct {
	account contracts.AccountInterface
}

// NewDepartment 创建 Department 实例。
func NewDepartment(account contracts.AccountInterface) *Department {
	return &Department{account: account}
}

// Create 创建部门
func (d *Department) Create(dept CreateDepartmentRequest) (int, error) {
	accessToken, err := d.account.AccessToken()
	if err != nil {
		return 0, kernelError.New(0, err)
	}

	apiUrl := fmt.Sprintf("cgi-bin/department/create?access_token=%s", accessToken.AccessToken)
	res, err := http.PostJSON(apiUrl, dept)
	if err != nil {
		return 0, kernelError.New(0, err)
	}

	var result createDepartmentResult
	err = json.Unmarshal(res, &result)
	if err != nil {
		return 0, kernelError.New(0, err)
	}
	if result.ErrCode != 0 {
		return 0, kernelError.NewWithApiError(result.ApiError)
	}

	return result.Id, nil
}

// Update 更新部门
func (d *Department) Update(dept UpdateDepartmentRequest) error {
	accessToken, err := d.account.AccessToken()
	if err != nil {
		return kernelError.New(0, err)
	}

	apiUrl := fmt.Sprintf("cgi-bin/department/update?access_token=%s", accessToken.AccessToken)
	res, err := http.PostJSON(apiUrl, dept)
	if err != nil {
		return kernelError.New(0, err)
	}

	var result kernelError.ApiError
	err = json.Unmarshal(res, &result)
	if err != nil {
		return kernelError.New(0, err)
	}
	if result.ErrCode != 0 {
		return kernelError.NewWithApiError(result)
	}

	return nil
}

// Delete 删除部门
func (d *Department) Delete(id int) error {
	accessToken, err := d.account.AccessToken()
	if err != nil {
		return kernelError.New(0, err)
	}

	apiUrl := fmt.Sprintf("cgi-bin/department/delete?access_token=%s&id=%d", accessToken.AccessToken, id)
	res, err := http.Get(apiUrl)
	if err != nil {
		return kernelError.New(0, err)
	}

	var result kernelError.ApiError
	err = json.Unmarshal(res, &result)
	if err != nil {
		return kernelError.New(0, err)
	}
	if result.ErrCode != 0 {
		return kernelError.NewWithApiError(result)
	}

	return nil
}

// List 获取部门列表
func (d *Department) List(id int) ([]DepartmentInfo, error) {
	accessToken, err := d.account.AccessToken()
	if err != nil {
		return nil, kernelError.New(0, err)
	}

	apiUrl := fmt.Sprintf("cgi-bin/department/list?access_token=%s", accessToken.AccessToken)
	if id > 0 {
		apiUrl = fmt.Sprintf("%s&id=%d", apiUrl, id)
	}
	res, err := http.Get(apiUrl)
	if err != nil {
		return nil, kernelError.New(0, err)
	}

	var result departmentListResult
	err = json.Unmarshal(res, &result)
	if err != nil {
		return nil, kernelError.New(0, err)
	}
	if result.ErrCode != 0 {
		return nil, kernelError.NewWithApiError(result.ApiError)
	}

	return result.Department, nil
}

// SimpleList 获取子部门ID列表
func (d *Department) SimpleList(id int) ([]DepartmentIdInfo, error) {
	accessToken, err := d.account.AccessToken()
	if err != nil {
		return nil, kernelError.New(0, err)
	}

	apiUrl := fmt.Sprintf("cgi-bin/department/simplelist?access_token=%s", accessToken.AccessToken)
	if id > 0 {
		apiUrl = fmt.Sprintf("%s&id=%d", apiUrl, id)
	}

	res, err := http.Get(apiUrl)
	if err != nil {
		return nil, kernelError.New(0, err)
	}

	var result departmentIdListResult
	err = json.Unmarshal(res, &result)
	if err != nil {
		return nil, kernelError.New(0, err)
	}
	if result.ErrCode != 0 {
		return nil, kernelError.NewWithApiError(result.ApiError)
	}

	return result.DepartmentId, nil
}

// Get 获取单个部门详情
func (d *Department) Get(id int) (*DepartmentInfo, error) {
	accessToken, err := d.account.AccessToken()
	if err != nil {
		return nil, kernelError.New(0, err)
	}

	apiUrl := fmt.Sprintf("cgi-bin/department/get?access_token=%s&id=%d", accessToken.AccessToken, id)
	res, err := http.Get(apiUrl)
	if err != nil {
		return nil, kernelError.New(0, err)
	}

	var result departmentGetResult
	err = json.Unmarshal(res, &result)
	if err != nil {
		return nil, kernelError.New(0, err)
	}
	if result.ErrCode != 0 {
		return nil, kernelError.NewWithApiError(result.ApiError)
	}

	return &result.Department, nil
}
