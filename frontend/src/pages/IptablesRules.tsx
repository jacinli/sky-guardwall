import { ReloadOutlined, SearchOutlined } from '@ant-design/icons'
import {
  Badge,
  Button,
  Card,
  Col,
  Input,
  Row,
  Select,
  Space,
  Statistic,
  Table,
  Tag,
  Tooltip,
  Typography,
  message,
} from 'antd'
import type { ColumnsType } from 'antd/es/table'
import dayjs from 'dayjs'
import { useCallback, useEffect, useState } from 'react'
import client from '../api/client'
import type { IptablesRule, IptablesRulesResponse } from '../types'

const { Text } = Typography

const TARGET_COLORS: Record<string, string> = {
  ACCEPT: 'green',
  DROP: 'red',
  REJECT: 'orange',
  RETURN: 'default',
  MASQUERADE: 'purple',
  DNAT: 'cyan',
  SNAT: 'blue',
}

const LINE_TYPE_LABELS: Record<string, string> = {
  policy: '策略',
  chain: '链',
  rule: '规则',
}

export default function IptablesRules() {
  const [data, setData] = useState<IptablesRulesResponse | null>(null)
  const [loading, setLoading] = useState(false)
  const [syncing, setSyncing] = useState(false)
  const [chainFilter, setChainFilter] = useState<string>('')
  const [searchText, setSearchText] = useState('')

  const fetchRules = useCallback(() => {
    setLoading(true)
    client
      .get('/iptables/rules', { params: { chain: chainFilter || undefined } })
      .then((res) => setData(res.data.data))
      .catch(() => message.error('获取规则失败'))
      .finally(() => setLoading(false))
  }, [chainFilter])

  useEffect(() => {
    fetchRules()
    const interval = setInterval(fetchRules, 60_000)
    return () => clearInterval(interval)
  }, [fetchRules])

  const handleSync = () => {
    setSyncing(true)
    client
      .post('/iptables/sync')
      .then(() => {
        message.success('同步完成')
        fetchRules()
      })
      .catch((err) => {
        const msg = err?.response?.data?.message ?? '同步失败'
        message.error(msg)
      })
      .finally(() => setSyncing(false))
  }

  const filteredRules = (data?.rules ?? []).filter((r) => {
    if (!searchText) return true
    const q = searchText.toLowerCase()
    return (
      r.chain.toLowerCase().includes(q) ||
      r.target.toLowerCase().includes(q) ||
      r.source.toLowerCase().includes(q) ||
      r.raw_line.toLowerCase().includes(q)
    )
  })

  const columns: ColumnsType<IptablesRule> = [
    {
      title: '链',
      dataIndex: 'chain',
      key: 'chain',
      width: 160,
      render: (v: string) => <Tag color="blue">{v}</Tag>,
    },
    {
      title: '类型',
      dataIndex: 'line_type',
      key: 'line_type',
      width: 80,
      render: (v: string) => (
        <Text type="secondary" style={{ fontSize: 12 }}>
          {LINE_TYPE_LABELS[v] ?? v}
        </Text>
      ),
    },
    {
      title: '来源 IP',
      dataIndex: 'source',
      key: 'source',
      width: 160,
      render: (v: string) => v || <Text type="secondary">any</Text>,
    },
    {
      title: '目标 IP',
      dataIndex: 'dest',
      key: 'dest',
      width: 160,
      render: (v: string) => v || <Text type="secondary">any</Text>,
    },
    {
      title: '协议',
      dataIndex: 'protocol',
      key: 'protocol',
      width: 80,
      render: (v: string) => v || <Text type="secondary">-</Text>,
    },
    {
      title: '端口',
      dataIndex: 'dst_port',
      key: 'dst_port',
      width: 80,
      render: (v: string) => v || <Text type="secondary">-</Text>,
    },
    {
      title: '动作',
      dataIndex: 'target',
      key: 'target',
      width: 100,
      render: (v: string) => (
        <Tag color={TARGET_COLORS[v] ?? 'default'}>{v || '-'}</Tag>
      ),
    },
    {
      title: '原始规则',
      dataIndex: 'raw_line',
      key: 'raw_line',
      ellipsis: { showTitle: false },
      render: (v: string) => (
        <Tooltip title={v} placement="topLeft">
          <Text code style={{ fontSize: 11 }}>
            {v}
          </Text>
        </Tooltip>
      ),
    },
  ]

  const counts = {
    total: data?.total ?? 0,
    chains: data?.chains?.length ?? 0,
    drop: (data?.rules ?? []).filter((r) => r.target === 'DROP').length,
    reject: (data?.rules ?? []).filter((r) => r.target === 'REJECT').length,
  }

  return (
    <Space direction="vertical" size={16} style={{ width: '100%' }}>
      {/* Stats */}
      <Row gutter={16}>
        {[
          { title: '规则总数', value: counts.total, color: '#1677ff' },
          { title: '链数量', value: counts.chains, color: '#52c41a' },
          { title: 'DROP 规则', value: counts.drop, color: '#f5222d' },
          { title: 'REJECT 规则', value: counts.reject, color: '#fa8c16' },
        ].map((s) => (
          <Col span={6} key={s.title}>
            <Card size="small">
              <Statistic title={s.title} value={s.value} valueStyle={{ color: s.color }} />
            </Card>
          </Col>
        ))}
      </Row>

      {/* Table */}
      <Card
        title="iptables 规则"
        extra={
          <Space>
            {data?.last_synced_at && (
              <Text type="secondary" style={{ fontSize: 12 }}>
                同步于 {dayjs(data.last_synced_at).format('HH:mm:ss')}
              </Text>
            )}
            {data?.sync_has_error && (
              <Badge status="error" text={<Text type="danger" style={{ fontSize: 12 }}>同步错误</Text>} />
            )}
            <Button
              icon={<ReloadOutlined spin={syncing} />}
              onClick={handleSync}
              loading={syncing}
              size="small"
            >
              立即同步
            </Button>
          </Space>
        }
      >
        <Space style={{ marginBottom: 12 }}>
          <Select
            placeholder="筛选链"
            allowClear
            style={{ width: 160 }}
            value={chainFilter || undefined}
            onChange={(v) => setChainFilter(v ?? '')}
            options={(data?.chains ?? []).map((c) => ({ label: c, value: c }))}
          />
          <Input
            placeholder="搜索规则..."
            prefix={<SearchOutlined />}
            value={searchText}
            onChange={(e) => setSearchText(e.target.value)}
            style={{ width: 220 }}
            allowClear
          />
        </Space>

        <Table
          rowKey="id"
          dataSource={filteredRules}
          columns={columns}
          loading={loading}
          size="small"
          scroll={{ x: 900 }}
          pagination={{ pageSize: 50, showSizeChanger: true, showTotal: (t) => `共 ${t} 条` }}
        />
      </Card>
    </Space>
  )
}
