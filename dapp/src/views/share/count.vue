<template>
  <f7-page name="count">
    <ChildrenHeader />
    <div class="share-main">
      <f7-toolbar class="tab-menu" bottom tabbar>
        <f7-link tab-link="#tab-a" tab-link-active>{{lang('我的节点')}}</f7-link>
        <f7-link tab-link="#tab-b">{{lang('我的收益')}}</f7-link>
        <f7-link tab-link="#tab-c">{{lang('矩阵图')}}</f7-link>
      </f7-toolbar>
      <f7-tabs animated>
        <f7-tab id="tab-a" class="tab-content" tab-active>
          <f7-block>
            <div class="node-head">
              <div class="node-head-item">
                <p>{{lang('级别')}}</p>
                <p>{{ levelType[userinfo.level] }}</p>
              </div>
              <div class="node-head-item">
                <p>{{lang('分享节点')}}</p>
                <p>{{ userinfo.locationNum }}</p>
              </div>
              <div class="node-head-item">
                <p>{{lang('总业绩')}}</p>
                <p>{{userinfo.total}}</p>
              </div>
              <div class="node-head-item">
                <p>{{lang('大区业绩')}}</p>
                <p>{{userinfo.max}}</p>
              </div>
              <div class="node-head-item">
                <p>{{lang('小区业绩')}}</p>
                <p>{{userinfo.min}}</p>
              </div>
              <div class="node-head-item">
                <p>{{lang('推荐人')}}</p>
                <p style="font-size: 13px">{{ formatAddress(userinfo.inviteUserAddress) }}</p>
              </div>
            </div>
            <!-- <div class="node-list">
              <div class="null-content" v-if="userinfo.listRecommend.length === 0">{{lang('暂无数据')}}</div>
              <ul v-else>
                <li v-for="(item, index) in userinfo.listRecommend" :key="index">{{ item.address }}</li>
              </ul>
            </div> -->
          </f7-block>
        </f7-tab>
        <f7-tab id="tab-b" class="tab-content">
          <f7-block>
            <div class="content-box">
              <div class="income-box">
                <div class="income-main">
                  <p>{{lang('待领取收益')}}</p>
                  <p>${{userinfo.amountGetSub || 0}}</p>
                </div>
                <div class="income-footer">
                  <div class="income-footer-item">
                    <p>{{lang('节点')}}</p>
                    <p>{{userinfo.buy || 0}}</p>
                  </div>
                  <div class="income-footer-item">
                    <p>{{lang('待产出')}}</p>
                    <p>{{userinfo.amountGetSub || 0}}</p>
                  </div>
                  <div class="income-footer-item">
                    <p>{{lang('已产量')}}</p>
                    <p>{{userinfo.amountGet || 0}}</p>
                  </div>
                  <div class="income-footer-item">
                    <p>{{lang('出局次数')}}</p>
                    <p>{{userinfo.outNum || 0}}</p>
                  </div>
                  <div class="income-footer-item">
                    <p>{{lang('静态收益')}}</p>
                    <p>{{userinfo.location}}</p>
                  </div>
                  <div class="income-footer-item">
                    <p>{{lang('直推收益')}}</p>
                    <p>{{userinfo.recommend}}</p>
                  </div>
                  <div class="income-footer-item">
                    <p>{{lang('直推加速')}}</p>
                    <p>{{userinfo.recommendTwo}}</p>
                  </div>
                  <div class="income-footer-item">
                    <p>{{lang('团队收益')}}</p>
                    <p>{{userinfo.team}}</p>
                  </div>
                  <div class="income-footer-item">
                    <p>{{lang('平级收益')}}</p>
                    <p>{{userinfo.teamTwo}}</p>
                  </div>
                  <div class="income-footer-item">
                    <p>{{lang('全网收益')}}</p>
                    <p>{{userinfo.all}}</p>
                  </div>
                </div>
              </div>
            </div>
            <div class="income-list">
              <div class="income-list-header">
                <ul class="header-list">
                  <li :class="{'active': activeTab === item[0]}" v-for="(item, index) in menuType" :key="index" @click="changeTab(item[0])">{{ item[1] }}</li>
                </ul>
              </div>
              <div class="income-list-main">
                <div class="income-list-item" v-for="(item, index) in rewardList" :key="index">
                  <div class="income-list-item-info">
                    <p>
                      <span v-if="!['8'].includes(activeTab)">usdt数量：{{ item.amount }}</span>
                      <span v-if="!['1'].includes(activeTab)">brc20数量：{{ item.amountTwo }}</span>
                    </p>
                    <p style="font-size: 13px;">
                      <span v-if="!['1', '2', '7', '8'].includes(activeTab)">{{ formatAddress(item.address) }}</span>
                      <span v-if="!['1', '2', '3', '7', '8'].includes(activeTab)">代数：{{ item.num }}</span>
                    </p>
                    <p>{{ item.createdAt }}</p>
                  </div>
                  <div class="income-list-item-money">{{ item.reward }}</div>
                </div>
                <div class="empty-data" v-if="rewardList.length === 0"></div>
                <Pagination
                  v-model="page"
                  :page-count="allPageCount"
                  mode="simple"
                  @change="getRewardList"
                />
              </div>
            </div>
          </f7-block>
        </f7-tab>
        <f7-tab id="tab-c" class="tab-content">
          <f7-block>
            <a-tree
              v-if="treeData.length > 0"
              v-model:expandedKeys="expandedKeys"
              v-model:selectedKeys="selectedKeys"
              :load-data="onLoadData"
              :tree-data="treeData"
            />
          </f7-block>
        </f7-tab>
      </f7-tabs>
    </div>
  </f7-page>
</template>
<script setup lang="ts">
import ChildrenHeader from '../../components/header/childrenHeader.vue'
import userPerson from "@/pinia/person";
import type { TreeProps } from 'ant-design-vue';
import lang from '@/i18n/index'
import { Pagination } from "vant";
import { f7, f7ready } from 'framework7-vue'
import { onMounted, onBeforeUnmount } from 'vue';
import request from "@/tools/request";
const person = userPerson();
const userinfo = $computed(() => person.userinfo);
const address = $computed(() => person.address);

let page = $ref(1);
let allPageCount = $ref(3);
let activeTab: string = $ref('1');
let pickerDevice: any = $ref(null);
let rewardList: any[] = $ref([]);

const menuType: any[] = [
  ['1', lang('认购')],
  ['2', lang('静态收益')],
  ['3', lang('直推收益')],
  ['4', lang('直推加速收益')],
  ['5', lang('团代收益')],
  ['6', lang('平级收益')],
  ['7', lang('全网收益')],
  ['8', lang('赠送') + 'brc20']
]

const getRewardList = async (page: number = 1) => {
  const res: any = await request.get("app_server/reward_list", {
    params: {
      page,
      reqType: activeTab
    }
  });

  allPageCount = Math.ceil(res.count / 10);
  rewardList = res.list
}

getRewardList()

const changeTab = (type: string) => {
  activeTab = type;

  page = 1
  getRewardList()
}

onMounted(() => {
})

onBeforeUnmount(() => {
})

const levelType: { [key: string]: string } = {
  '-1': lang('未激活'),
  '0': lang('节点'),
  '1': 'v1',
  '2': 'v2',
  '3': 'v3',
  '4': 'v4',
  '5': 'v5'
}

const expandedKeys = $ref<string[]>([]);
const selectedKeys = $ref<string[]>([]);
let treeData: any = $ref([]);

const collectKeys = (nodes: any[], out: any[] = []) => {
  (nodes || []).forEach((n) => {
    out.push(n.key)
    if (n.children && n.children.length) collectKeys(n.children, out)
  })
  return out
}

const mapRecommend = (item: any, key: any, full: boolean) => {
  const kids = (item.children || []).map((c: any, i: number) => mapRecommend(c, `${key}-${i}`, full))
  return {
    title: `${formatAddress(item.address)}---(${lang('数量')}:${item.amount})`,
    key: String(key),
    amount: item.amount,
    address: item.address,
    isLeaf: kids.length === 0 && !!full,
    children: kids.length ? kids : (full ? [] : undefined)
  }
}

const onLoadData: TreeProps['loadData'] = (treeNode: any) => {
  return new Promise<void>(async (resolve) => {
    if (treeNode.dataRef.children) {
      resolve();
      return;
    }

    const res: any = await request.get(`app_server/recommend_list?address=${treeNode.dataRef.address}`);

    // res.area = [
    //   {
    //     address: formatAddress("aaaa12378123hjdfowis88883"),
    //     locationId: "3",
    //     countLow: 2,
    //   },
    //   {
    //     address: formatAddress("32423998f8uijkjkrejkw2223"),
    //     locationId: "4",
    //     countLow: 0
    //   }
    // ]
    setTimeout(() => {
      const full = !!res.full
      treeNode.dataRef.children = (res.recommends || []).map((item: any, index: number) => mapRecommend(item, `${treeNode.eventKey}-${index}`, full))
      treeData = [...treeData];
      resolve();
    }, 1000);
  });
};

const formatAddress = (value: string) => {
  const frontSix = value.slice(0, 6);
  const backSix = value.slice(-4);
  const middle = '...';
  return frontSix + middle + backSix;
}

const getUserArea = async () => {
  const res: any = await request.get(`app_server/recommend_list?address=${address}`);
  // res.area = [
  //   {
  //     address: formatAddress("aaaa12378123hjdfowis88883"),
  //     locationId: "3",
  //     countLow: 2,
  //   },
  //   {
  //     address: formatAddress("32423998f8uijkjkrejkw2223"),
  //     locationId: "4",
  //     countLow: 0
  //   }
  // ]
  const full = !!res.full
  treeData = (res.recommends || []).map((item: any, index: number) => mapRecommend(item, index, full))
  if (full) {
    expandedKeys = collectKeys(treeData)
  }
}

getUserArea()


</script>
<style scoped lang="less">
@import "./styles/index.less";
</style>
