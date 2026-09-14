<template>
<div class='shipping-address'>
  <van-nav-bar
    :title="lang('收货地址')"
    left-arrow
    :border="false"
    fixed
    @click-left="handleClick"
  />
  <div class="address-form" v-if="editStatus">
    <van-form @submit="onSubmit">
      <van-cell-group inset>
        <van-field
          v-model="name"
          name="name"
          :label="lang('姓名')"
          :placeholder="lang('请填写姓名')"
          :rules="[{ required: true, message: lang('请填写姓名') }]"
        />
        <van-field
          v-model="phone"
          name="phone"
          :label="lang('手机号')"
          :placeholder="lang('请填写手机号')"
          :rules="[{ required: true, message: lang('请填写手机号') }]"
        />
        <van-field
          v-model="country"
          name="country"
          :label="lang('国家')"
          :placeholder="lang('请填写国家')"
          :rules="[{ required: true, message: lang('请填写国家') }]"
        />
        <van-field
          v-model="province"
          name="province"
          :label="lang('省份')"
          :placeholder="lang('请填写省份')"
          :rules="[{ required: true, message: lang('请填写省份') }]"
        />
        <van-field
          v-model="city"
          name="city"
          :label="lang('城市')"
          :placeholder="lang('请填写城市')"
          :rules="[{ required: true, message: lang('请填写城市') }]"
        />
        <van-field
          v-model="area"
          name="area"
          :label="lang('地区')"
          :placeholder="lang('请填写地区')"
          :rules="[{ required: true, message: lang('请填写地区') }]"
        />
        <van-field
          v-model="detail"
          rows="3"
          name="detail"
          :label="lang('详情')"
          :placeholder="lang('请填写详情')"
          type="textarea"
          :rules="[{ required: true, message: lang('请填写详情') }]"
        />
      </van-cell-group>
      <div style="margin: 16px; display: flex; gap: 12px;">
        <van-button
          icon="delete-o"
          type="danger"
          plain
          @click="onDelete"
          v-if="editStatus === 'revise'"
        />
        <van-button round style="flex: 1" @click="closeEdit">
          {{ lang('取消') }}
        </van-button>
        <van-button round type="primary" style="flex: 1" native-type="submit">
          {{ lang('保存') }}
        </van-button>
      </div>
    </van-form>
  </div>
  <van-address-list
    v-else
    v-model="chosenAddressId"
    :list="addressList"
    :switchable="false"
    :default-tag-text="lang('默认')"
    :add-button-text="lang('新增地址')"
    @add="onAdd"
    @edit="onEdit"
  >
  </van-address-list>
</div>
</template>
<script setup>
import userPerson from "@/pinia/person";
import lang from '@/i18n/index'
import request from "@/tools/request";
import { showToast, showLoadingToast, showConfirmDialog, closeToast, showFailToast, showDialog, closeDialog, showSuccessToast } from "vant";
import { ref } from 'vue'

const person = userPerson();
const userinfo = $computed(() => person.userinfo);
const addressList = $computed(() => {
  return person.userinfo.userAddress.map((item, index) => {
    return {
      id: item.id,
      name: item.name,
      tel: item.phone,
      address: item.country + item.province + item.city + item.area + item.detail,
      isDefault: index === 0,
      originData: item
    }
  })
})

const sign = $computed(() => person.sign);
const chosenAddressId = $ref('1');
const editStatus = $ref(null)
const id = $ref('')
const name = $ref('')
const phone = $ref('')
const country = $ref('')
const province = $ref('')
const city = $ref('')
const area = $ref('')
const detail = $ref('')
const onSubmit = async () => {
  await request.post("app_server/set_address", {
    name,
    phone,
    country,
    province,
    city,
    area,
    detail,
    sign
  }).then((res) => {
    if (res.status === 'ok') {
      person.getUser()
      showSuccessToast(lang('添加成功'))
      editStatus = null
    } else {
      showFailToast(res.status)
    }
  })
}

const onAdd = () => {
  id = ''
  name = ''
  phone = ''
  country = ''
  province = ''
  city = ''
  area = ''
  detail = ''
  editStatus = 'create'
};

const onDelete = async () => {
  await request.post("app_server/delete_address", {
    id,
    sign
  }).then((res) => {
    if (res.status === 'ok') {
      person.getUser()
      showSuccessToast(lang('删除成功'))
      editStatus = null
    } else {
      showFailToast(res.status)
    }
  })
}

const onEdit = (item, index) => {
  const data = addressList[index].originData

  id = data.id
  name = data.name
  phone = data.phone
  country = data.country
  province = data.province
  city = data.city
  area = data.area
  detail = data.detail

  editStatus = 'revise'
};

const closeEdit = () => {
  editStatus = null
}


const props = defineProps({
  changeShippingAddress: {
    type: Function,
    required: true
  }
})

function handleClick() {
  editStatus = null
  props.changeShippingAddress(false)
}
</script>
<style lang='less' scoped>
.shipping-address {
  padding-top: 56px;
}
.address-form {
  border-radius: 6px;
}
</style>