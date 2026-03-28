import { DeleteOutlined, PlusOutlined } from '@ant-design/icons'
import {
  Badge,
  Button,
  Card,
  Col,
  Form,
  Input,
  InputNumber,
  Popconfirm,
  Row,
  Select,
  Space,
  Table,
  Tag,
  Typography,
  message,
} from 'antd'
import type { ColumnsType } from 'antd/es/table'
import dayjs from 'dayjs'
import { useEffect, useState } from 'react'
import client from '../api/client'
import type { AddRuleRequest, ManagedRule } from '../types'

const { Text } = Typography

const TARGET_COLORS: Record<string, string> = {
  ACCEPT: 'green',
  DROP: 'red',
  REJECT: 'orange',
}

export default function ManagedRules() {
  const [rules, setRules] = useState<ManagedRule[]>([])
  const [loading, setLoading] = useState(false)
  const [submitting, setSubmitting] = useState(false)
  const [form] = Form.useForm<AddRuleRequest>()

  const fetchRules = () => {
    setLoading(true)
    client
      .get('/managed-rules')
      .then((res) => setRules(res.data.data.rules ?? []))
      .catch(() => message.error('获取规则失败'))
      .finally(() => setLoading(false))
  }

  useEffect(() => {
    fetchRules()
  }, [])

  const handleAdd = async (values: AddRuleRequest) => {
    setSubmitting(true)
    try {
      await client.post('/managed-rules', {
        ...values,
        dst_port: values.dst_port ?? 0,
        src_ip: values.src_ip ?? '',
        description: values.description ?? '',
      })
      message.success('规则添加成功并已写入 iptables')
      form.resetFields()
      fetchRules()
    } catch (err: unknown) {
      const msg =
        (err as { response?: { data?: { message?: string } } })?.response?.data?.message ??
        '添加失败'
      message.error(msg)
    } finally {
      setSubmitting(false)
    }
  }

  const handleDelete = async (id: number) => {
    try {
      await client.delete(`/managed-rules/${id}`)
      message.success('规则已删除')
      fetchRules()
    } catch (err: unknown) {
      const msg =
        (err as { response?: { data?: { message?: string } } })?.response?.data?.message ??
        '删除失败'
      message.error(msg)
    }
  }

  const columns: ColumnsType<ManagedRule> = [
    {
      title: '说明',
      dataIndex: 'description',
      key: 'description',
      render: (v: string) => v || <Text type="secondary">-</Text>,
    },
    {
      title: '链',
      dataIndex: 'chain',
      key: 'chain',
      width: 90,
      render: (v: string) => <Tag color="blue">{v}</Tag>,
    },
    {
      title: '来源 IP',
      dataIndex: 'src_ip',
      key: 'src_ip',
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
      render: (v: number) => (v > 0 ? v : <Text type="secondary">any</Text>),
    },
    {
      title: '动作',
      dataIndex: 'target',
      key: 'target',
      width: 90,
      render: (v: string) => <Tag color={TARGET_COLORS[v] ?? 'default'}>{v}</Tag>,
    },
    {
      title: '状态',
      dataIndex: 'is_applied',
      key: 'is_applied',
      width: 90,
      render: (v: boolean) => (
        <Badge status={v ? 'success' : 'default'} text={v ? '已应用' : '未应用'} />
      ),
    },
    {
      title: '创建时间',
      dataIndex: 'created_at',
      key: 'created_at',
      width: 160,
      render: (v: string) => dayjs(v).format('MM-DD HH:mm:ss'),
    },
    {
      title: '操作',
      key: 'action',
      width: 80,
      render: (_, record) => (
        <Popconfirm
          title="确认删除？"
          description="将同时从 iptables 移除此规则"
          okText="删除"
          cancelText="取消"
          okButtonProps={{ danger: true }}
          onConfirm={() => handleDelete(record.id)}
        >
          <Button type="text" danger icon={<DeleteOutlined />} size="small" />
        </Popconfirm>
      ),
    },
  ]

  return (
    <Space direction="vertical" size={16} style={{ width: '100%' }}>
      <Row gutter={16}>
        {/* Add rule form */}
        <Col xs={24} lg={8}>
          <Card title={<><PlusOutlined /> 添加防火墙规则</>}>
            <Form
              form={form}
              layout="vertical"
              onFinish={handleAdd}
              initialValues={{ chain: 'INPUT', protocol: 'tcp', target: 'DROP' }}
            >
              <Form.Item label="说明（可选）" name="description">
                <Input placeholder="例：封禁扫描 IP" maxLength={200} />
              </Form.Item>

              <Form.Item label="链" name="chain" rules={[{ required: true }]}>
                <Select
                  options={[
                    { label: 'INPUT', value: 'INPUT' },
                    { label: 'OUTPUT', value: 'OUTPUT' },
                    { label: 'FORWARD', value: 'FORWARD' },
                  ]}
                />
              </Form.Item>

              <Form.Item
                label="来源 IP / CIDR（可选，空=任意）"
                name="src_ip"
                rules={[
                  {
                    pattern: /^(\d{1,3}\.){3}\d{1,3}(\/\d{1,2})?$|^$/,
                    message: '请输入合法的 IP 或 CIDR',
                  },
                ]}
              >
                <Input placeholder="例：1.2.3.4 或 10.0.0.0/8" />
              </Form.Item>

              <Row gutter={8}>
                <Col span={12}>
                  <Form.Item label="协议" name="protocol" rules={[{ required: true }]}>
                    <Select
                      options={[
                        { label: 'tcp', value: 'tcp' },
                        { label: 'udp', value: 'udp' },
                        { label: 'icmp', value: 'icmp' },
                        { label: 'all', value: 'all' },
                      ]}
                    />
                  </Form.Item>
                </Col>
                <Col span={12}>
                  <Form.Item label="目标端口（0=任意）" name="dst_port">
                    <InputNumber min={0} max={65535} style={{ width: '100%' }} placeholder="0" />
                  </Form.Item>
                </Col>
              </Row>

              <Form.Item label="动作" name="target" rules={[{ required: true }]}>
                <Select
                  options={[
                    { label: 'DROP（静默丢弃）', value: 'DROP' },
                    { label: 'REJECT（拒绝+回复）', value: 'REJECT' },
                    { label: 'ACCEPT（允许）', value: 'ACCEPT' },
                  ]}
                />
              </Form.Item>

              <Form.Item style={{ marginBottom: 0 }}>
                <Button type="primary" htmlType="submit" loading={submitting} block>
                  立即写入 iptables
                </Button>
              </Form.Item>
            </Form>
          </Card>
        </Col>

        {/* Rules table */}
        <Col xs={24} lg={16}>
          <Card
            title="已管理的规则"
            extra={
              <Button size="small" onClick={fetchRules} loading={loading}>
                刷新
              </Button>
            }
          >
            <Table
              rowKey="id"
              dataSource={rules}
              columns={columns}
              loading={loading}
              size="small"
              pagination={{ pageSize: 20, showTotal: (t) => `共 ${t} 条` }}
              locale={{ emptyText: '暂无自定义规则' }}
            />
          </Card>
        </Col>
      </Row>
    </Space>
  )
}
