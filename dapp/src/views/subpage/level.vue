<template>
<div class='shop-page'>
  <van-nav-bar
    :title="lang('等级权益')"
    left-arrow
    :border="false"
    fixed
    @click-left="handleBack"
  />
  <div class="page-main">
    <i class="jiangbei"></i>
    <div class="page-title">
      <p>{{ lang('普通会员权益') }}</p>
    </div>
    <div class="box-1">
      <span>{{ lang('分享奖励加速比例') }}</span>
      <span>10%</span>
    </div>
    <div class="box-2">
      <div class="box-2-item">
        <p>{{ userinfo.locationNum }}</p>
        <p>{{ lang('推荐人数') }}</p>
      </div>
      <div class="box-2-item">
        <p>10</p>
        <p>{{ lang('享受代数') }}</p>
      </div>
    </div>
    <div class="box-1">
      <span>{{ lang('奖励比例') }}(%)</span>
      <span>10,9,8,7,6,5,4,3,2,1</span>
    </div>
    <div class="content-text-1">{{ lang('当前团队等级') }}：V{{ userinfo.level }}</div>
    <div class="box-3">
      <div class="box-3-title">V{{ userinfo.level }} {{ lang('权益') }}</div>
      <ul class="box-3-list">
        <li><i class="i-1-icon" />{{ lang('获得团队静态收益的') }}16.00% {{ lang('加速') }}</li>
        <li><i class="i-2-icon" />{{ lang('获得下级同级用户收益的') }}10.00% {{ lang('加速') }}</li>
        <li><i class="i-3-icon" />{{ lang('获得全网每日质押总业绩') }}1.00%{{ lang('分红加速') }}</li>
        <li><i class="i-4-icon" />{{ lang('首次达标该等级赠送') }}ISPAY10000.00{{ lang('个') }}</li>
      </ul>
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

</script>
<style lang='less' scoped>
  .shop-page {
    min-height: 100vh;
    .page-main {
      width: 100%;
      min-height: 50vh;
      padding: 15px 15px 80px 15px;
      box-sizing: border-box;
      position: relative;
      .jiangbei {
        width: 130px;
        height: 135px;
        background: url('../../assets/images/jiangbei.png') no-repeat;
        background-size: 130px 135px;
        position: absolute;
        right: 0;
        top: 10px;
        z-index: 11;
      }
      .page-title {
        padding: 60px 0 30px 0;
        h2 {
          font-size: 16px;
        }
        p {
          line-height: 1.6;
        }
      }
      .box-1 {
        padding: 26px 20px;
        color: #fff;
        background: #383729;
        border: 1px solid #83703b;
        border-radius: 12px;
        padding: 15px 30px 15px 15px;
        align-items: center;
        // background-image: url(../../assets/images/topbg1.png);
        background-repeat: no-repeat;
        background-size: 100% 249px;
        display: flex;
        justify-content: space-between;
        margin-bottom: 20px;
        span {
          &:nth-child(2) {
            font-size: 18px;
            color: rgb(255, 209, 39);
          }
        }
      }
      .box-2 {
        display: flex;
        justify-content: space-between;
        background: hsla(0, 0%, 100%, .05);
        border: 1px solid #666;
        border-radius: 18px;
        margin-bottom: 20px;
        .box-2-item {
          width: 50%;
          height: 88px;
          display: flex;
          flex-direction: column;
          justify-content: center;
          align-items: center;
          gap: 5px;
          p {
            font-size: 14px;
            color: #CCC;
            margin: 0;
            &:nth-child(1) {
              font-size: 20px;
              color: #fff;
            }
          }
        }
      }
      .content-text-1 {
        margin-bottom: 20px;
      }
      .box-3 {
        margin-bottom: 20px;
        .box-3-title {
          padding: 0 0 26px 0px;
          color: #fff;
          text-align: center;
        }
        .box-3-list {
          background: hsla(0, 0%, 100%, .05);
          border: 1px solid #666;
          border-radius: 18px;
          padding: 20px 26px;
          color: #fff;
          font-size: 14px;
          font-weight: 400;
          display: flex;
          flex-direction: column;
          gap: 15px;
          li {
            display: flex;
            align-items: center;
            gap: 5px;
            i {
              width: 22px;
              height: 22px;
              display: inline-block;
              &.i-1-icon {
                background: url('../../assets/images/i-1.png');
                background-size: 22px 22px;
              }
              &.i-2-icon {
                background: url('../../assets/images/i-2.png');
                background-size: 22px 22px;
              }
              &.i-3-icon {
                background: url('../../assets/images/i-3.png');
                background-size: 22px 22px;
              }
              &.i-4-icon {
                background: url('../../assets/images/i-4.png');
                background-size: 22px 22px;
              }
            }
          }
        }
      }
    }
  }
</style>