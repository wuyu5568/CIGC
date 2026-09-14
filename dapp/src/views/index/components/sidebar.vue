<template>
<div class='sidebar-box'>
  <div class="sidebar-logo"></div>
  <div class="user-address">{{lang('地址')}}：{{ formatAddress(address) }}</div>
  <ul class="sidebar-nav">
    <li><i class="nav-icon-1" />{{ lang('首页') }}</li>
    <li @click="router.push('recharge')"><i class="nav-icon-10" />{{ lang('充值') }}</li>
    <li @click="router.push('transfer')"><i class="nav-icon-11" />{{ lang('转账') }}</li>
    <li @click="router.push('shop')"><i class="nav-icon-2" />{{ lang('股份商店') }}</li>
    <li @click="router.push('powerShop')"><i class="nav-icon-3" />{{ lang('新预售2') }}</li>
    <li @click="router.push('Web3Shop')"><i class="nav-icon-3" />Web3 {{ lang('商城') }}</li>
    <li @click="router.push('stat')"><i class="nav-icon-4" />{{ lang('算力统计') }}</li>
    <li @click="router.push('wallet')"><i class="nav-icon-5" />{{ lang('我的资产') }}</li>
    <!-- <li @click="router.push('level')"><i class="nav-icon-6" />{{ lang('等级权益') }}</li> -->
    <li @click="router.push('community')"><i class="nav-icon-7" />{{ lang('社群建设') }}</li>
    <li @click="openLink('https://www.playgamefrane.com/game/')"><i class="nav-icon-8" />{{ lang('魔晶农场') }}</li>
    <li @click="showToast(lang('暂未开通'))"><i class="nav-icon-9" />{{ lang('资产跨链') }}</li>
    <li @click="showToast(lang('暂未开通'))"><i class="nav-icon-10" />{{ lang('主网钱包') }}</li>
  </ul>
</div>
</template>
<script setup>
import { showToast } from 'vant'
import userPerson from "@/pinia/person";
import { useRouter } from 'vue-router'
import { ref, computed } from 'vue'
import lang from '@/i18n/index'

const router = useRouter()
const person = userPerson();
const address = computed(() => person.address);

const formatAddress = (value) => {
  const frontSix = value.slice(0, 6);
  const backSix = value.slice(-6);
  const middle = '...';
  return frontSix + middle + backSix;
}

const openLink = (url) => {
  window.location.href = url
}

const openEmpty = () => {
  showToast(lang('暂未开通'))
}

</script>
<style lang='less' scoped>
.sidebar-box {
  width: 168px;
  height: 100vh;
  padding: 0 15px;
  position: relative;
  z-index: 999;
  overflow-y: auto;
  font-size: 15px;
  color: #fff;
  background: #171c21;
  background-image: url(../../../assets/images/topbg.png);
  background-repeat: no-repeat;
  background-size: 100% 254px;
  display: flex;
  flex-direction: column;
  align-items: center;
  .sidebar-logo {
    width: 72px;
    height: 72px;
    background: url('../../../assets/images/logo.png') no-repeat;
    background-size: 72px 72px;
    margin: 50px 0 20px 0;
    flex-shrink: 0;
  }
  .user-address {
    font-size: 13px;
    margin-bottom: 20px;
    flex-shrink: 0;
  }
  .sidebar-nav {
    width: 100%;
    display: flex;
    flex-direction: column;
    text-align: center;
    gap: 10px;
    li {
      width: 100%;
      line-height: 36px;
      background: url('data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAAAB4AAAAeCAYAAAA7MK6iAAACPUlEQVR4AcTWO0jDUBQG4L4Gh7aTg6ODg459jA4WHDo4KDgouAgKjjqIg4o6OAiCio6KCIKDSAWXSoXspY+p6NbRrUNbaKEv/7/klgi29+am0pLDPbc058tN0pN4XCP6KMHpdHrCMIyxYR6jFM5ms/sej+c7GAx+5vP5yWHhUhjQAsLV6XQm2+22MSxcCrvd7kPAdUQXb7VaKZ56zp2EFA6Hwx/Al4A0EdymcOqNTCYzzoluSGEWjkQiSWAryAU+jTxRKBT8GLU2JZiVsfIXr9e7wdyM2Vqt9qZ7tyvDxEKh0APwbeZmzPn9/oQObgsmBvwK1/yAOQN5XAe3DRPDNT8FeMycgTweCASesXIf5yqhBbMw8BOAPRzfLQB/UsW1YUAuEz9nbsYy8GszHzg4glkZ+C5WfsPcjC202SMz7zs4hlm50WicYex2N4zcum2WSb9wDLOD+Xy+FADx9OIB9O56fP/n5gg2O1cCldnJMLia6HBrOP3vnAwKbbhYLI7V63WiswIAusUOJ+aDRi2YaKlUSuBROW8pvg30zjIfmNqGLWhcVMZdfRyNRq/EXGW0BbM5YKVPWOkvFNf0RAWz/sYWjNefe6CLogBXqoNyf2UYTeEC6Bp3YgC91EW5vxIM9Aho73EI9BHoDgvohhTO5XJ7QK0Pg9dyubyuC4r9pDDeLHvtDweQrFQqq7FYTLwCiTq2RymM03qKql+I22q1ugSULRFTZ5sUxrVM4j86g9gcFspDlsL80X/EyOAfAAAA//8UKYQFAAAABklEQVQDAIaIwD0C5FPuAAAAAElFTkSuQmCC') no-repeat right center;
      background-size: 16px 16px;
      cursor: pointer;
      position: relative;
      i {
        width: 22px;
        height: 22px;
        display: block;
        position: absolute;
        left: 0;
        top: 50%;
        transform: translateY(-50%);
        &.nav-icon-1 {
          background: url('../../../assets/images/icon/icon1.png') no-repeat;
          background-size: 22px 22px;
        }
        &.nav-icon-2 {
          background: url('../../../assets/images/icon/icon2.png') no-repeat;
          background-size: 22px 22px;
        }
        &.nav-icon-3 {
          background: url('../../../assets/images/icon/icon3.png') no-repeat;
          background-size: 22px 22px;
        }
        &.nav-icon-4 {
          background: url('../../../assets/images/icon/icon4.png') no-repeat;
          background-size: 22px 22px;
        }
        &.nav-icon-5 {
          background: url('../../../assets/images/icon/icon5.png') no-repeat;
          background-size: 22px 22px;
        }
        &.nav-icon-6 {
          background: url('../../../assets/images/icon/icon6.png') no-repeat;
          background-size: 22px 22px;
        }
        &.nav-icon-7 {
          background: url('../../../assets/images/icon/icon7.png') no-repeat;
          background-size: 22px 22px;
        }
        &.nav-icon-8 {
          background: url('../../../assets/images/icon/icon8.png') no-repeat;
          background-size: 22px 22px;
        }
        &.nav-icon-9 {
          background: url('../../../assets/images/icon/icon9.png') no-repeat;
          background-size: 22px 22px;
        }
        &.nav-icon-10 {
          background: url('../../../assets/images/icon/icon10.png') no-repeat;
          background-size: 22px 22px;
        }
        &.nav-icon-11 {
          background: url('../../../assets/images/icon/icon11.png') no-repeat;
          background-size: 22px 22px;
        }
      }
    }
  }
}
</style>