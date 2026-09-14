<template>
  <div class="node-page">
    <ChildrenHeader />
    <div class="staking">
      <div class="staking-banner"></div>
      <div class="staking-amount">
        <div class="staking-amount-item">
          <div class="staking-amount-title">isPay {{lang('余额')}}</div>
          <div class="staking-amount-number">0.00</div>
        </div>
        <div class="staking-amount-item">
          <div class="staking-amount-title">KEY {{lang('余额')}}</div>
          <div class="staking-amount-number">0.00</div>
        </div>
      </div>
      <div class="staking-box">
        <div class="staking-box-head">
          <div class="staking-head-title">
            <i class="staking-head-title-avatar"></i>isPay({{lang('永续')}})
          </div>
          <button class="staking-button" @click="showToastTop">{{lang('金库')}}</button>
        </div>
        <div class="staking-box-content content-line">
          <div class="staking-box-content-item">
            <div class="staking-box-content-data" style="color: #F9D02A">KEY</div>
            <div class="staking-box-content-title">{{lang('赚取')}}</div>
          </div>
          <div class="staking-box-content-item">
            <div class="staking-box-content-data" style="color: #F9D02A">0/s</div>
            <div class="staking-box-content-title">{{lang('产出速度')}}</div>
          </div>
        </div>
        <div class="staking-box-content">
          <div class="staking-box-content-item">
            <div class="staking-box-content-data" style="color: #F9D02A">isPay</div>
            <div class="staking-box-content-title">{{lang('赚取')}}</div>
          </div>
          <div class="staking-box-content-item">
            <div class="staking-box-content-data" style="color: #F9D02A">0/s</div>
            <div class="staking-box-content-title">{{lang('产出速度')}}</div>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>
<script setup lang="ts">
import ChildrenHeader from '../../components/header/childrenHeader.vue'
import lang from '@/i18n/index'
import userPerson from "@/pinia/person";
import { showToast } from 'vant'
import { ref, computed } from 'vue'

const person = userPerson();
const userinfo = computed(() => person.userinfo);

const showToastTop = () => {
  showToast({
    message: lang('暂未开放该功能'),
    position: 'top',
    duration: 2000,
  });
}
</script>
<style scoped lang="less">
@import "./styles/index.less";

.node-page {
  width: 100%;
  min-height: 100vh;
}

.staking-button {
  padding: 5px 15px;
  background-color: #F9D02A;
  color: #000;
  border: none;
  border-radius: 4px;
  cursor: pointer;
  font-size: 14px;
  font-weight: bold;
}
</style>
