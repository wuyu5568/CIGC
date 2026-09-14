<template>
<div class='address'>
  <div class="address-editor">
    <div class="address-editor-item">
      <van-field v-model="name" required :label="lang('收件人')" />
    </div>
    <div class="address-editor-item">
      <van-field v-model="phone" required type="tel" :label="lang('手机号')" />
    </div>
    <div class="address-editor-item">
      <van-field
        v-model="national"
        required
        is-link
        readonly
        :label="lang('国家')"
        :placeholder="lang('请选择国家')"
        @click="showPicker = true"
      />
      <van-popup v-model:show="showPicker" destroy-on-close round position="bottom">
        <van-picker
          :model-value="pickerValue"
          :columns="columns"
          @cancel="showPicker = false"
          @confirm="onConfirm"
        />
      </van-popup>
    </div>
    <div class="address-editor-item">
      <van-field v-model="address" required type="tel" :label="lang('详细地址')" />
    </div>
    <button class="add-address" :disabled="doubleInfo" @click="saveAddress">{{lang('保存')}}</button>
  </div>

</div>
</template>
<script setup lang="ts">
import lang from '@/i18n/index'
import { useI18n } from 'vue-i18n'
import { showToast, showSuccessToast, showFailToast } from 'vant';
import countries from 'i18n-iso-countries';
import en from 'i18n-iso-countries/langs/en.json';
import zh from 'i18n-iso-countries/langs/zh.json';
import request from "@/tools/request";
import userPerson from "@/pinia/person";
const { locale } = useI18n()

interface PickerOption {
  text: string
  value: string | number
}

// Vant Picker confirm 回调参数类型
interface PickerConfirmEvent {
  selectedValues: (string | number)[]
  selectedOptions: PickerOption[]
}

countries.registerLocale(locale.value === 'zh' ? zh : en);
const countryMap = countries.getNames(locale.value, { select: 'official' });
const sign = $computed(() => person.sign);

if (locale.value !== 'zh') {
  countryMap.CN = 'China'
}

const person = userPerson();
const userinfo = $computed(() => person.userinfo);

let name = $ref(userinfo.seven || '');
let phone = $ref(userinfo.six || '');
let address = $ref(userinfo.five || '');
let national = $ref(userinfo.one || '')

const columns = Object.values(countryMap).map(value => ({
  text: value,
  value: value
}));

const doubleInfo = $computed(() => {
  if (userinfo.seven === name && userinfo.six === phone && userinfo.five === address && userinfo.one === national) {
    return true
  } else {
    return false
  }
})

// 控制 Picker 显示隐藏
let showPicker = $ref(false)

// 当前选择的值
let pickerValue: Array<number|string> = $ref([])

// 确认选择
const onConfirm = ({ selectedValues, selectedOptions }: PickerConfirmEvent): void => {
  showPicker = false
  pickerValue = selectedValues
  console.log('selectedOptions', selectedOptions)
  national = selectedOptions[0].value as string
}

const saveAddress = async () => {
  if (!name) return showToast(lang('请填写收件人'));
  if (!phone) return showToast(lang('请填写手机号'));
  if (!national) return showToast(lang('请选择国家'));
  if (!address) return showToast(lang('请填写详细地址'));

  let res: any = await request.post('app_server/set_info', {
    one: national,
    five: address,
    six: phone,
    seven: name,
    sign
  })

  if (res.status === 'ok') {
    showSuccessToast(lang('保存成功'));
    person.getUser();
  } else {
    showFailToast(res.status);
  }
}

</script>
<style lang='less' scoped>
.no-address {
  text-align: center;
  display: flex;
  flex-direction: column;
  align-items: center;
  background: #29313C;
  border-radius: 6px;
  padding: 50px 30px 30px 30px;
  gap: 20px;
  p {
    color: #999;
    margin-bottom: 20px;
  }
  .add-address {
    width: 100px;
    height: 32px;
    line-height: 32px;
    background: #F8B204;
    border: 0;
    border-radius: 4px;
    display: inline-block;
    font-size: 13px;
    opacity: .9;
    font-weight: 500;
  }
}
.address-editor {
  text-align: center;
  display: flex;
  flex-direction: column;
  align-items: flex-start;
  background: #29313C;
  border-radius: 6px;
  padding: 10px 10px 30px 10px;
  gap: 5px;
  h3 {
    width: 100%;
    font-size: 14px;
    margin-bottom: 20px;
    text-align: center;
  }
  .address-editor-item {
    width: 100%;
    display: flex;
    --van-cell-background: #2A2A2A;
    .address-label {
      padding: 10px 0;
      padding-left: 15px;
    }
  }
  .add-address {
    width: 100px;
    height: 32px;
    line-height: 32px;
    background: #F8B204;
    border: 0;
    border-radius: 4px;
    display: inline-block;
    font-size: 13px;
    opacity: .9;
    font-weight: 500;
    margin: 30px auto 0 auto;
    &:disabled {
      background: #555;
    }
  }
}
</style>