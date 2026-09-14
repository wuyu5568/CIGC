<template>
  <PageView>
      <a-card title="列表">
          <a-row :gutter="10" class="inputGroup">
              <a-col :xs="12" :md="6" :lg="6" :xl="4">
                  <a-input v-model="searchData.address" placeholder="账户" @keyup.enter="getListTwo" />
              </a-col>
              <a-col :xs="12" :md="6" :lg="6" :xl="4">
                  <a-button-group>
                      <a-button type="primary" :loading="loading" @click="getListTwo">确定筛选</a-button>
                  </a-button-group>
              </a-col>
          </a-row>
          <a-table :loading="loading" :columns="columns" :dataSource="data" :pagination="{ total, pageSize, current }"
              @change="changePagination" bordered :scroll="{ x: true }">
          </a-table>
      </a-card>
  </PageView>
</template>

<script type="text/jsx">
import Gai from '../../api/Gai'
import listMixin from '../mixin/listMixin'
export default {
  name: 'investor',
  mixins: [listMixin],
  data() {
      return {
          columns: [
              {
                  title: '账户',
                  dataIndex: 'address',
              },
              {
                  title: '挖矿',
                  dataIndex: 'current',
              },
              {
                  title: '节点',
                  dataIndex: 'usdt',
              },
              {
                  title: '收益',
                  dataIndex: 'currentMaxSub',
              },
              {
                  title: '出局',
                  dataIndex: 'out',
              }
          ],
          searchData: {
              address: '',
          },
      }
  },
  methods: {
      getList() {
          this.loading = true
          Gai.location_list({
              page: this.current,
              status: 'running',
              ...this.searchData
          }).then((res) => {
              this.data = res.locations.map((value, key) => {
                  return { ...value, key }
              })
              this.loading = false
              this.total = parseInt(res.count)
          })
      },
  },
}
</script>

<style scoped lang="less">
.inputGroup {
  >div {
      margin-bottom: 20px;
  }
}
</style>