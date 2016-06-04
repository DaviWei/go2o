/**
 * Copyright 2015 @ z3q.net.
 * name : category_manager.go
 * author : jarryliu
 * date : 2016-06-04 13:40
 * description :
 * history :
 */
package sale

import (
	"bytes"
	"fmt"
	"go2o/core/domain/interface/sale"
	"go2o/core/domain/interface/valueobject"
	"go2o/core/infrastructure/domain"
	"sort"
	"strconv"
	"strings"
	"time"
)

var _ sale.ICategory = new(categoryImpl)

// 分类实现
type categoryImpl struct {
	_value           *sale.Category
	_rep             sale.ICategoryRep
	_parentIdChanged bool
	_childIdArr      []int
	_opt             domain.IOptionStore
}

func newCategory(rep sale.ICategoryRep, v *sale.Category) sale.ICategory {
	return &categoryImpl{
		_value: v,
		_rep:   rep,
	}
}

func (this *categoryImpl) GetDomainId() int {
	return this._value.Id
}

func (this *categoryImpl) GetValue() *sale.Category {
	return this._value
}

func (this *categoryImpl) GetOption() domain.IOptionStore {
	if this._opt == nil {
		opt := newCategoryOption(this)
		if err := opt.Stat(); err != nil {
			opt.Set(sale.C_OptionViewName, &domain.Option{
				Key:   sale.C_OptionViewName,
				Type:  domain.OptionTypeString,
				Must:  false,
				Title: "显示页面",
				Value: "goods_list.html",
			})
			opt.Set(sale.C_OptionDescribe, &domain.Option{
				Key:   sale.C_OptionDescribe,
				Type:  domain.OptionTypeString,
				Must:  false,
				Title: "描述",
				Value: "",
			})
			opt.Flush()
		}
		this._opt = opt
	}
	return this._opt
}

func (this *categoryImpl) SetValue(v *sale.Category) error {
	val := this._value
	if val.Id == v.Id {
		val.Description = v.Description
		val.Enabled = v.Enabled
		val.Name = v.Name
		val.SortNumber = v.SortNumber
		val.Icon = v.Icon
		if val.ParentId != v.ParentId {
			this._parentIdChanged = true
			val.ParentId = v.ParentId
		} else {
			this._parentIdChanged = false
		}
	}
	return nil
}

// 获取子栏目的编号
func (this *categoryImpl) GetChildId() []int {
	if this._childIdArr == nil {
		childCats := this._rep.GetChildCategories(this._value.MerchantId, this.GetDomainId())
		this._childIdArr = make([]int, len(childCats))
		for i, v := range childCats {
			this._childIdArr[i] = v.Id
		}
	}
	return this._childIdArr
}

func (this *categoryImpl) Save() (int, error) {
	id, err := this._rep.SaveCategory(this._value)
	if err == nil {
		this._value.Id = id
		if len(this._value.Url) == 0 || (this._parentIdChanged &&
			strings.HasPrefix(this._value.Url, "/c-")) {
			this._value.Url = this.getAutomaticUrl(this._value.MerchantId, id)
			this._parentIdChanged = false
			return this.Save()
		}
	}
	return id, err
}

func (this *categoryImpl) getAutomaticUrl(merchantId, id int) string {
	var relCategories []*sale.Category = this._rep.GetRelationCategories(merchantId, id)
	var buf *bytes.Buffer = bytes.NewBufferString("/c")
	var l int = len(relCategories)
	for i := l; i > 0; i-- {
		buf.WriteString("-" + strconv.Itoa(relCategories[i-1].Id))
	}
	buf.WriteString(".htm")
	return buf.String()
}

var _ domain.IOptionStore = new(categoryOption)

// 分类数据选项
type categoryOption struct {
	domain.IOptionStore
	_mchId int
	_c     *categoryImpl
}

func newCategoryOption(c *categoryImpl) domain.IOptionStore {
	var file string
	mchId := c.GetValue().MerchantId
	if mchId > 0 {
		file = fmt.Sprintf("conf/mch/%d/option/c/%d", mchId, c.GetDomainId())
	} else {
		file = fmt.Sprintf("conf/core/sale/cate_opt_%d", c.GetDomainId())
	}
	return &categoryOption{
		_mchId:       c.GetValue().ParentId,
		_c:           c,
		IOptionStore: domain.NewOptionStoreWrapper(file),
	}
}

var _ sale.ICategoryManager = new(categoryManagerImpl)

//当商户共享系统的分类时,没有修改的权限,既只读!
type categoryManagerImpl struct {
	_readonly   bool
	_rep        sale.ICategoryRep
	_valRep     valueobject.IValueRep
	_mchId      int
	_categories []sale.ICategory
}

func NewCategoryManager(mchId int, rep sale.ICategoryRep,
	valRep valueobject.IValueRep) sale.ICategoryManager {
	c := &categoryManagerImpl{
		_rep:   rep,
		_mchId: mchId,
	}
	return c.init()
}

func (this *categoryManagerImpl) init() sale.ICategoryManager {
	return this
}

// 获取栏目关联的编号,系统用0表示
func (this *categoryManagerImpl) getRelationId() int {
	return this._mchId
}

// 是否只读,当商户共享系统的分类时,
// 没有修改的权限,即只读!
func (this *categoryManagerImpl) ReadOnly() bool {
	return this._readonly
}

// 创建分类
func (this *categoryManagerImpl) CreateCategory(v *sale.Category) sale.ICategory {
	if v.CreateTime == 0 {
		v.CreateTime = time.Now().Unix()
	}
	v.MerchantId = this.getRelationId()
	return newCategory(this._rep, v)
}

// 获取分类
func (this *categoryManagerImpl) GetCategory(id int) sale.ICategory {
	v := this._rep.GetCategory(this.getRelationId(), id)
	if v != nil {
		return this.CreateCategory(v)
	}
	return nil
}

// 获取所有分类
func (this *categoryManagerImpl) GetCategories() []sale.ICategory {
	//if this.categories == nil {
	list := this._rep.GetCategories(this.getRelationId())
	sort.Sort(list)
	this._categories = make([]sale.ICategory, len(list))
	for i, v := range list {
		this._categories[i] = this.CreateCategory(v)
	}
	//}
	return this._categories
}

// 删除分类
func (this *categoryManagerImpl) DeleteCategory(id int) error {
	//todo: 删除应放到这里来处理
	return this._rep.DeleteCategory(this.getRelationId(), id)
}