export default {
    data () {
        return {
            loading: true,
            selectedRowKeys: [],
            data: [],
            total: 0,
            pageSize: 10,
            current: 1,
            showSizeChanger: true
        }
    },
    activated () {
        this.getList()
    },
    methods: {
        getListTwo () {
            this.current = 1
            this.getList()
        },
        changePagination (pagination) {
            Object.keys(pagination).forEach(val => {
                this[val] = pagination[val]
            })
            this.getList()
        },
        onSelectChange (selectedRowKeys) {
            this.selectedRowKeys = selectedRowKeys
        }
    }
}