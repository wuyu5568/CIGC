<template>
    <!-- :right-text="lang('规则')" -->
<div class='page'>
  <van-nav-bar
    :title="lang('算力统计')"
    left-arrow
    :border="false"
    fixed
    @click-left="handleBack"
    @click-right="handleRule"
  />
  <div class="page-main">
    <div class="current-price">
      <span>{{ lang('当前代币价格') }}</span>
      <span>1.000000 USDT</span>
    </div>
    <div class="page-title">{{ lang('算力统计') }}</div>
    <div class="total-power">
      <span>{{ lang('我的总算力') }}</span>
      <span>0.00 MB(0.00%)</span>
    </div>
    <div class="double-column">
      <div class="column-left">
        <p>{{ lang('我的原发算力') }}</p>
        <p>0.00 MB</p>
      </div>
      <div class="column-right">
        <p>{{ lang('我的分享算力') }}</p>
        <p>0.00 MB</p>
      </div>
    </div>
    <div class="current-price">
      <span>{{ lang('USD Brc20') }}</span>
      <p>{{ userinfo.rawNew }}</p>
    </div>
    <div class="double-column-2">
      <div class="column-left">
        <p>{{ lang('我的溢出算力') }}</p>
        <p>0.00 MB</p>
      </div>
      <div class="column-right">
        <p>{{ lang('全网原发算力') }}</p>
        <p>0.00 MB</p>
      </div>
    </div>
    <div class="double-column-3">
      <div class="column-left">
        <p>{{ lang('全网分享算力') }}</p>
        <p>0.00 MB</p>
      </div>
      <div class="column-right">
        <p>{{ lang('全网总算力') }}</p>
        <p>0.00 MB</p>
      </div>
    </div>
  </div>
</div>
</template>
<script setup>
import userPerson from "@/pinia/person";
import { useRouter } from 'vue-router'
import lang from '@/i18n/index'

const router = useRouter()
const person = userPerson();
const userinfo = $computed(() => person.userinfo);

const handleBack = () => {
  router.back()
}

const handleRule = () => {

}

</script>
<style lang='less' scoped>
  .page {
    min-height: 100vh;
    box-sizing: border-box;
    padding: 50px 15px 20px 15px;
    background: url('../../assets/images/topbg4.png') no-repeat;
    background-size: 100% auto;
    .current-price {
      background: hsla(0, 0%, 100%, .1);
      border: 1px solid #666;
      border-radius: 8px;
      padding: 20px 15px;
      display: flex;
      margin-bottom: 20px;
      justify-content: space-between;
      span {
        &:nth-child(2) {
          color: rgb(255, 209, 39);
          font-weight: 500;
        }
      }
    }
    .page-title {
      padding: 20px 0;
      font-weight: bold;
    }
    .total-power {
      width: 100%;
      height: 89px;
      background: url('../../assets/images/boxbg.png') no-repeat;
      background-size: 100% 89px;
      display: flex;
      justify-content: space-between;
      align-items: center;
      padding: 0 20px;
      box-sizing: border-box;
      margin-bottom: 20px;
      span {
        &:nth-child(2) {
          font-size: 16px;
          font-weight: bold;
        }
      }
    }
    .double-column {
      display: flex;
      justify-content: space-between;
      gap: 10px;
      margin-bottom: 20px;
      &>div {
        flex: 1;
        border-radius: 8px;
        padding: 15px;
        display: flex;
        flex-direction: column;
        gap: 15px;
        p {
          &:nth-child(2) {
            font-size: 16px;
            font-weight: 500;
          }
        }
      }
      .column-left {
        background: #ffd127;
        color: #171c21;
      }
      .column-right {
        background: #3640f0;
      }
    }
    .double-column-2 {
      display: flex;
      justify-content: space-between;
      gap: 10px;
      margin-bottom: 20px;
      &>div {
        border-radius: 8px;
        padding: 15px;
        display: flex;
        flex-direction: column;
        gap: 15px;
        p {
          &:nth-child(2) {
            font-size: 16px;
            font-weight: 500;
          }
        }
      }
      .column-left {
        width: 45%;
        background: #3640f0;
        color: #FFF;
      }
      .column-right {
        flex: 1;
        background: #171c21;
        border: 1px solid #3640f0;
      }
    }
    .double-column-3 {
      display: flex;
      justify-content: space-between;
      gap: 10px;
      margin-bottom: 20px;
      &>div {
        border-radius: 8px;
        padding: 15px;
        display: flex;
        flex-direction: column;
        gap: 15px;
        p {
          &:nth-child(2) {
            font-size: 16px;
            font-weight: 500;
          }
        }
      }
      .column-left {
        width: 50%;
        background: #171c21;
        border: 1px solid #ffd127;
        color: #FFF
      }
      .column-right {
        flex: 1;
        background: #ffd127;
        color: #222;
      }
    }
  }
</style>