<template>
  <div class="recharge-page">
    <ChildrenHeader />
    <div class="recharge-box">
      <!-- 标签栏 -->
      <div class="tab-title">
        <div class="tab-link" :class="{'tab-link-active': tabMenu === 1}" @click="tabMenu = 1">USDT{{ lang('充值') }}</div>
        <!-- <div class="tab-link" :class="{'tab-link-active': tabMenu === 2}" @click="tabMenu = 2">BIW{{ lang('充值') }}</div> -->
      </div>
      
      <!-- 标签内容 -->
      <div class="tabs-container">
        <!-- 标签A内容 -->
        <div class="tab-content" v-if="tabMenu === 1">
          <div class="block">
            <img class="recharge-code" :src="usdtCode" />
            <p class="recharge-text">{{ userinfo.address }}</p>
            <button class="copy-address-btn" @click="copyToClipboard(userinfo.address)">{{ lang('复制地址') }}</button>
          </div>
        </div>
        
        <!-- 标签B内容 -->
        <div class="tab-content" v-if="tabMenu === 2">
          <div class="block">
            <img class="recharge-code" :src="biwCode" />
            <p class="recharge-text">{{ userinfo.addressBiw }}</p>
            <button class="copy-address-btn" @click="copyToClipboard(userinfo.addressBiw)">{{ lang('复制地址') }}</button>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>
<script setup lang="ts">
import ChildrenHeader from '../../components/header/childrenHeader.vue'
import QRCode from 'qrcode'
import userPerson from "@/pinia/person";
import lang from '@/i18n/index'
import copy from 'copy-to-clipboard';
import { showToast } from 'vant'
import { ref, computed, onMounted } from 'vue'

const person = userPerson();
const userinfo = computed(() => person.userinfo);

let tabMenu = ref(1)
let code = ref('')
let usdtCode = ref('')
let biwCode = ref('')

const getQRCodePic = (text: string) => {
  return new Promise((resolve, reject) => {
    QRCode.toDataURL(text, (err: any, url: any) => {
      if (err) {
        reject(err);
        return;
      }
      resolve(url);
    });
  })
}

const initQRCode = async () => {
  if (userinfo.value.address) {
    usdtCode.value = await getQRCodePic(userinfo.value.address) as string
  }
  if (userinfo.value.addressBiw) {
    biwCode.value = await getQRCodePic(userinfo.value.addressBiw) as string
  }
}

onMounted(() => {
  initQRCode()
})

const copyToClipboard = (text: string) => {
  copy(text);
  showToast({
    message: lang("地址已复制到剪贴板"),
    position: 'center',
    duration: 2000,
  });
}

</script>
<style scoped lang="less">
.recharge-page {
  width: 100%;
  min-height: 100vh;
}

.recharge-box {
  padding: 20px;
  border-radius: 10px;
  margin: 10px 5%;
}

/* 标签栏样式 */
.tab-title {
  display: flex;
  justify-content: space-around;
  align-items: center;
  background-color: #29313C;
  padding: 10px 0;
  margin-bottom: 20px;
}

.tab-link {
  color: #CCC;
  padding: 10px 20px;
  cursor: pointer;
  transition: all 0.3s;
}

.tab-link-active {
  color: #FCD434;
  font-weight: bold;
}

/* 标签内容容器 */
.tabs-container {
  margin-bottom: 20px;
}

.tab-content {
  padding: 10px;
}

/* 块样式 */
.block {
  padding: 10px;
  background-color: #333;
  border-radius: 8px;
  margin-bottom: 10px;
  text-align: center;
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.recharge-code {
  width: 200px;
  margin: 0 auto;
}

.recharge-text {
  overflow-wrap: break-word;
  text-align: center;
}

.copy-address-btn {
  width: 120px;
  height: 36px;
  border-radius: 4px;
  margin: 0 auto;
  border: 0;
  background: @c7;
  color: #FFF;
  cursor: pointer;
}
</style>
