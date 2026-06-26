#!/usr/bin/env node
/**
 * 账易会计系统 - 28个行业科目模板生成器
 * 生成完整的科目模板JSON文件
 */

const fs = require('fs');
const path = require('path');

const BASE = path.join(__dirname, 'v2');

// ============================================================
// Helper: 构建科目
// ============================================================
function acc(code, name, direction, level, aux = []) {
  return { code, name, direction, level, aux };
}

// ============================================================
// 一级科目（通用）
// ============================================================
function getBaseAccounts(standard) {
  const isSB = standard === 'small_business';
  const accts = [
    // 资产类
    acc('1001', '库存现金', '借', 1),
    acc('1002', '银行存款', '借', 1),
    acc('1012', '其他货币资金', '借', 1),
    acc('1101', isSB ? '短期投资' : '交易性金融资产', '借', 1),
    acc('1121', '应收票据', '借', 1),
    acc('1122', '应收账款', '借', 1, ['customer']),
    acc('1123', '预付账款', '借', 1, ['supplier']),
    acc('1131', '应收股利', '借', 1),
    acc('1132', '应收利息', '借', 1),
    acc('1221', '其他应收款', '借', 1, ['employee']),
    acc('1401', '材料采购', '借', 1),
    acc('1402', '在途物资', '借', 1),
    acc('1403', '原材料', '借', 1),
  ];

  if (!isSB) {
    accts.push(acc('1404', '材料成本差异', '借', 1));
  }

  accts.push(acc('1405', '库存商品', '借', 1));

  if (!isSB) {
    accts.push(acc('1406', '发出商品', '借', 1));
  }

  accts.push(acc('1407', '商品进销差价', '借', 1));
  accts.push(acc('1408', '委托加工物资', '借', 1));
  accts.push(acc('1411', '周转材料', '借', 1));

  if (!isSB) {
    accts.push(acc('1471', '存货跌价准备', '借', 1));
  }

  accts.push(acc('1501', isSB ? '长期债券投资' : '持有至到期投资', '借', 1));
  accts.push(acc('1511', '长期股权投资', '借', 1));
  accts.push(acc('1601', '固定资产', '借', 1));
  accts.push(acc('1602', '累计折旧', '贷', 1));
  acc('1604', '在建工程', '借', 1) && accts.push(acc('1604', '在建工程', '借', 1));
  accts.push(acc('1605', '工程物资', '借', 1));
  accts.push(acc('1606', '固定资产清理', '借', 1));
  accts.push(acc('1701', '无形资产', '借', 1));
  accts.push(acc('1702', '累计摊销', '贷', 1));

  if (!isSB) {
    accts.push(acc('1703', '无形资产减值准备', '贷', 1));
  }

  accts.push(acc('1801', '长期待摊费用', '借', 1));
  accts.push(acc('1901', '待处理财产损溢', '借', 1));

  // 负债类
  accts.push(acc('2001', '短期借款', '贷', 1));
  accts.push(acc('2201', '应付票据', '贷', 1));
  accts.push(acc('2202', '应付账款', '贷', 1, ['supplier']));
  accts.push(acc('2203', '预收账款', '贷', 1, ['customer']));
  accts.push(acc('2211', '应付职工薪酬', '贷', 1));
  accts.push(acc('2221', '应交税费', '贷', 1));
  accts.push(acc('2231', '应付利息', '贷', 1));
  accts.push(acc('2232', '应付股利', '贷', 1));
  accts.push(acc('2241', '其他应付款', '贷', 1));
  accts.push(acc('2401', '递延收益', '贷', 1));
  accts.push(acc('2501', '长期借款', '贷', 1));
  accts.push(acc('2701', '长期应付款', '贷', 1));

  // 所有者权益类
  accts.push(acc('3001', '实收资本', '贷', 1));
  accts.push(acc('3002', '资本公积', '贷', 1));
  accts.push(acc('3101', '盈余公积', '贷', 1));
  accts.push(acc('3103', '本年利润', '贷', 1));
  accts.push(acc('3104', '利润分配', '贷', 1));

  // 损益类
  accts.push(acc('5001', '主营业务收入', '贷', 1, ['customer', 'department']));
  accts.push(acc('5051', '其他业务收入', '贷', 1));
  accts.push(acc('5111', '投资收益', '贷', 1));
  accts.push(acc('5301', '营业外收入', '贷', 1));
  accts.push(acc('5401', '主营业务成本', '借', 1, ['department']));
  accts.push(acc('5402', '其他业务成本', '借', 1));
  accts.push(acc('5403', '营业税金及附加', '借', 1));
  accts.push(acc('5601', '销售费用', '借', 1, ['department']));
  accts.push(acc('5602', '管理费用', '借', 1, ['department']));
  accts.push(acc('5603', '财务费用', '借', 1));
  accts.push(acc('5711', '营业外支出', '借', 1));
  accts.push(acc('5801', '所得税费用', '借', 1));

  return accts;
}

// ============================================================
// 企业准则特殊科目
// ============================================================
function getEnterpriseOnlyAccounts() {
  return [
    acc('4301', '研发支出', '借', 1),
  ];
}

// ============================================================
// 应交税费明细 - 一般纳税人
// ============================================================
function getTaxDetailGeneral() {
  return [
    acc('2221.01', '应交增值税', '贷', 2),
    acc('2221.01.01', '进项税额', '借', 3),
    acc('2221.01.02', '销项税额', '贷', 3),
    acc('2221.01.03', '进项税额转出', '贷', 3),
    acc('2221.01.04', '转出未交增值税', '贷', 3),
    acc('2221.01.05', '转出多交增值税', '借', 3),
    acc('2221.01.06', '减免税款', '借', 3),
    acc('2221.02', '未交增值税', '贷', 2),
    acc('2221.03', '预交增值税', '借', 2),
    acc('2221.04', '待抵扣进项税额', '借', 2),
    acc('2221.05', '待认证进项税额', '借', 2),
    acc('2221.06', '增值税留抵税额', '借', 2),
    acc('2221.07', '简易计税', '贷', 2),
    acc('2221.08', '转让金融商品应交增值税', '贷', 2),
    acc('2221.09', '代扣代交增值税', '贷', 2),
    acc('2221.10', '应交消费税', '贷', 2),
    acc('2221.11', '应交企业所得税', '贷', 2),
    acc('2221.12', '应交个人所得税(代扣代缴)', '贷', 2),
    acc('2221.13', '应交城市维护建设税', '贷', 2),
    acc('2221.14', '应交教育费附加', '贷', 2),
    acc('2221.15', '应交地方教育附加', '贷', 2),
    acc('2221.16', '应交印花税', '贷', 2),
    acc('2221.17', '应交房产税', '贷', 2),
    acc('2221.18', '应交城镇土地使用税', '贷', 2),
    acc('2221.19', '应交车船税', '贷', 2),
    acc('2221.20', '应交环境保护税', '贷', 2),
  ];
}

// ============================================================
// 应交税费明细 - 小规模纳税人
// ============================================================
function getTaxDetailSmall() {
  return [
    acc('2221.01', '应交增值税', '贷', 2),
    acc('2221.02', '应交消费税', '贷', 2),
    acc('2221.03', '应交企业所得税', '贷', 2),
    acc('2221.04', '应交个人所得税(代扣代缴)', '贷', 2),
    acc('2221.05', '应交城市维护建设税', '贷', 2),
    acc('2221.06', '应交教育费附加', '贷', 2),
    acc('2221.07', '应交地方教育附加', '贷', 2),
    acc('2221.08', '应交印花税', '贷', 2),
    acc('2221.09', '应交房产税', '贷', 2),
    acc('2221.10', '应交城镇土地使用税', '贷', 2),
    acc('2221.11', '应交车船税', '贷', 2),
    acc('2221.12', '应交环境保护税', '贷', 2),
  ];
}

// ============================================================
// 行业特殊科目
// ============================================================

function getManufacturingAccounts(taxpayer, standard) {
  const isSB = standard === 'small_business';
  const accts = [
    // 成本类
    acc('4001', '生产成本', '借', 1),
    acc('4001.01', '直接材料', '借', 2),
    acc('4001.02', '直接人工', '借', 2),
    acc('4001.03', '制造费用', '借', 2),
    acc('4101', '制造费用', '借', 1),
    acc('4101.01', '折旧费', '借', 2, ['department']),
    acc('4101.02', '修理费', '借', 2, ['department']),
    acc('4101.03', '水电费', '借', 2, ['department']),
    acc('4101.04', '物料消耗', '借', 2, ['department']),
    acc('4101.05', '职工薪酬', '借', 2, ['department']),
    acc('4101.06', '其他', '借', 2, ['department']),
    // 材料采购明细
    acc('1401.01', '买价', '借', 2),
    acc('1401.02', '采购费用', '借', 2),
    // 原材料明细
    acc('1403.01', '主要材料', '借', 2, ['warehouse']),
    acc('1403.02', '辅助材料', '借', 2, ['warehouse']),
    acc('1403.03', '外购半成品', '借', 2, ['warehouse']),
    // 库存商品明细
    acc('1405.01', '产成品', '借', 2, ['warehouse']),
    acc('1405.02', '半成品', '借', 2, ['warehouse']),
    // 主营业务成本明细
    acc('5401.01', '产品销售成本', '借', 2, ['department']),
    // 销售费用明细
    acc('5601.01', '运输费', '借', 2, ['department']),
    acc('5601.02', '广告费', '借', 2, ['department']),
    acc('5601.03', '销售人员薪酬', '借', 2, ['department']),
    // 管理费用明细
    acc('5602.01', '管理人员薪酬', '借', 2, ['department']),
    acc('5602.02', '办公费', '借', 2, ['department']),
    acc('5602.03', '折旧费', '借', 2, ['department']),
    acc('5602.04', '修理费', '借', 2, ['department']),
    acc('5602.05', '水电费', '借', 2, ['department']),
    acc('5602.06', '差旅费', '借', 2, ['department']),
    acc('5602.07', '业务招待费', '借', 2, ['department']),
    acc('5602.08', '车辆使用费', '借', 2, ['department']),
    acc('5602.09', '其他', '借', 2, ['department']),
  ];

  if (!isSB) {
    accts.push(acc('4301', '研发支出', '借', 1));
    accts.push(acc('4301.01', '费用化支出', '借', 2));
    accts.push(acc('4301.02', '资本化支出', '借', 2));
  }

  return accts;
}

function getRetailAccounts(taxpayer, standard) {
  const isSB = standard === 'small_business';
  const accts = [
    // 库存商品按品类
    acc('1405.01', '商品类别一', '借', 2, ['warehouse']),
    acc('1405.02', '商品类别二', '借', 2, ['warehouse']),
    acc('1405.03', '商品类别三', '借', 2, ['warehouse']),
    // 商品进销差价
    acc('1407.01', '商品进销差价', '贷', 2),
    // 主营业务成本
    acc('5401.01', '商品销售成本', '借', 2, ['department']),
    // 销售费用
    acc('5601.01', '运输费', '借', 2, ['department']),
    acc('5601.02', '广告费', '借', 2, ['department']),
    acc('5601.03', '销售人员薪酬', '借', 2, ['department']),
    // 管理费用
    acc('5602.01', '管理人员薪酬', '借', 2, ['department']),
    acc('5602.02', '办公费', '借', 2, ['department']),
    acc('5602.03', '折旧费', '借', 2, ['department']),
    acc('5602.04', '修理费', '借', 2, ['department']),
    acc('5602.05', '水电费', '借', 2, ['department']),
    acc('5602.06', '差旅费', '借', 2, ['department']),
    acc('5602.07', '业务招待费', '借', 2, ['department']),
    acc('5602.08', '车辆使用费', '借', 2, ['department']),
    acc('5602.09', '其他', '借', 2, ['department']),
  ];

  if (!isSB) {
    accts.push(acc('4301', '研发支出', '借', 1));
  }

  return accts;
}

function getServiceAccounts(taxpayer, standard) {
  const isSB = standard === 'small_business';
  const accts = [
    // 主营业务成本
    acc('5401.01', '服务成本', '借', 2, ['department', 'project']),
    // 销售费用
    acc('5601.01', '广告宣传费', '借', 2, ['department']),
    acc('5601.02', '业务推广费', '借', 2, ['department']),
    acc('5601.03', '销售人员薪酬', '借', 2, ['department']),
    // 管理费用
    acc('5602.01', '管理人员薪酬', '借', 2, ['department']),
    acc('5602.02', '办公费', '借', 2, ['department']),
    acc('5602.03', '折旧费', '借', 2, ['department']),
    acc('5602.04', '修理费', '借', 2, ['department']),
    acc('5602.05', '水电费', '借', 2, ['department']),
    acc('5602.06', '差旅费', '借', 2, ['department']),
    acc('5602.07', '业务招待费', '借', 2, ['department']),
    acc('5602.08', '咨询顾问费', '借', 2, ['department']),
    acc('5602.09', '其他', '借', 2, ['department']),
  ];

  if (!isSB) {
    accts.push(acc('4301', '研发支出', '借', 1));
    accts.push(acc('4301.01', '费用化支出', '借', 2));
    accts.push(acc('4301.02', '资本化支出', '借', 2));
  }

  return accts;
}

function getConstructionAccounts(taxpayer, standard) {
  const isSB = standard === 'small_business';
  const accts = [
    // 工程施工
    acc('4401', '工程施工', '借', 1),
    acc('4401.01', '合同成本', '借', 2, ['project']),
    acc('4401.02', '间接费用', '借', 2, ['project']),
    // 机械作业
    acc('4403', '机械作业', '借', 1),
    acc('4403.01', '折旧费', '借', 2),
    acc('4403.02', '燃料及动力', '借', 2),
    acc('4403.03', '人工费', '借', 2),
    acc('4403.04', '其他', '借', 2),
    // 原材料明细
    acc('1403.01', '钢材', '借', 2, ['warehouse']),
    acc('1403.02', '水泥', '借', 2, ['warehouse']),
    acc('1403.03', '砂石', '借', 2, ['warehouse']),
    acc('1403.04', '其他材料', '借', 2, ['warehouse']),
    // 预收账款明细
    acc('2203.01', '工程预收款', '贷', 2, ['customer', 'project']),
    // 应收账款明细
    acc('1122.01', '工程结算款', '借', 2, ['customer', 'project']),
    // 主营业务成本
    acc('5401.01', '工程结算成本', '借', 2, ['project']),
    // 管理费用
    acc('5602.01', '管理人员薪酬', '借', 2, ['department']),
    acc('5602.02', '办公费', '借', 2, ['department']),
    acc('5602.03', '折旧费', '借', 2, ['department']),
    acc('5602.04', '差旅费', '借', 2, ['department']),
    acc('5602.05', '业务招待费', '借', 2, ['department']),
    acc('5602.06', '其他', '借', 2, ['department']),
  ];

  if (!isSB) {
    accts.push(acc('4301', '研发支出', '借', 1));
  }

  return accts;
}

function getRealEstateAccounts(taxpayer, standard) {
  const isSB = standard === 'small_business';
  const accts = [
    // 开发成本
    acc('4001', '开发成本', '借', 1),
    acc('4001.01', '土地征用及拆迁补偿费', '借', 2, ['project']),
    acc('4001.02', '前期工程费', '借', 2, ['project']),
    acc('4001.03', '建筑安装工程费', '借', 2, ['project']),
    acc('4001.04', '基础设施费', '借', 2, ['project']),
    acc('4001.05', '配套设施费', '借', 2, ['project']),
    acc('4001.06', '开发间接费', '借', 2, ['project']),
    // 开发产品
    acc('1402', '开发产品', '借', 1),
    acc('1402.01', '住宅', '借', 2, ['warehouse']),
    acc('1402.02', '商铺', '借', 2, ['warehouse']),
    acc('1402.03', '车位', '借', 2, ['warehouse']),
    // 预收账款明细
    acc('2203.01', '预售房款', '贷', 2, ['customer', 'project']),
    // 主营业务成本
    acc('5401.01', '开发产品销售成本', '借', 2, ['project']),
    // 管理费用
    acc('5602.01', '管理人员薪酬', '借', 2, ['department']),
    acc('5602.02', '办公费', '借', 2, ['department']),
    acc('5602.03', '折旧费', '借', 2, ['department']),
    acc('5602.04', '差旅费', '借', 2, ['department']),
    acc('5602.05', '业务招待费', '借', 2, ['department']),
    acc('5602.06', '销售费用', '借', 2, ['department']),
    acc('5602.07', '其他', '借', 2, ['department']),
  ];

  // 土地增值税
  if (taxpayer === 'general') {
    accts.push(acc('2221.21', '应交土地增值税', '贷', 2));
  } else {
    accts.push(acc('2221.13', '应交土地增值税', '贷', 2));
  }

  if (!isSB) {
    accts.push(acc('4301', '研发支出', '借', 1));
  }

  return accts;
}

function getTransportAccounts(taxpayer, standard) {
  const isSB = standard === 'small_business';
  const accts = [
    // 运输成本
    acc('4001', '运输成本', '借', 1),
    acc('4001.01', '燃油费', '借', 2, ['department']),
    acc('4001.02', '折旧费', '借', 2, ['department']),
    acc('4001.03', '人工费', '借', 2, ['department']),
    acc('4001.04', '维修费', '借', 2, ['department']),
    acc('4001.05', '过路过桥费', '借', 2, ['department']),
    acc('4001.06', '保险费', '借', 2, ['department']),
    // 固定资产明细
    acc('1601.01', '车辆', '借', 2),
    acc('1601.02', '船舶', '借', 2),
    acc('1601.03', '飞机', '借', 2),
    acc('1601.04', '其他运输设备', '借', 2),
    // 主营业务成本
    acc('5401.01', '运输服务成本', '借', 2, ['department']),
    // 销售费用
    acc('5601.01', '业务推广费', '借', 2, ['department']),
    acc('5601.02', '销售人员薪酬', '借', 2, ['department']),
    acc('5601.03', '其他', '借', 2, ['department']),
    // 管理费用
    acc('5602.01', '管理人员薪酬', '借', 2, ['department']),
    acc('5602.02', '办公费', '借', 2, ['department']),
    acc('5602.03', '折旧费', '借', 2, ['department']),
    acc('5602.04', '差旅费', '借', 2, ['department']),
    acc('5602.05', '业务招待费', '借', 2, ['department']),
    acc('5602.06', '其他', '借', 2, ['department']),
  ];

  if (!isSB) {
    accts.push(acc('4301', '研发支出', '借', 1));
  }

  return accts;
}

function getAgricultureAccounts(taxpayer, standard) {
  const isSB = standard === 'small_business';
  const accts = [
    // 农业生产成本
    acc('4001', '农业生产成本', '借', 1),
    acc('4001.01', '种子', '借', 2, ['department']),
    acc('4001.02', '肥料', '借', 2, ['department']),
    acc('4001.03', '农药', '借', 2, ['department']),
    acc('4001.04', '人工费', '借', 2, ['department']),
    acc('4001.05', '机械作业费', '借', 2, ['department']),
    // 农业制造费用
    acc('4101', '农业制造费用', '借', 1),
    acc('4101.01', '折旧费', '借', 2, ['department']),
    acc('4101.02', '水电费', '借', 2, ['department']),
    acc('4101.03', '修理费', '借', 2, ['department']),
    acc('4101.04', '其他', '借', 2, ['department']),
    // 生物资产
    acc('1601', '生物资产', '借', 1),
    acc('1601.01', '消耗性生物资产', '借', 2),
    acc('1601.02', '生产性生物资产', '借', 2),
    // 生产性生物资产累计折旧
    acc('1602', '生产性生物资产累计折旧', '贷', 1),
    // 原材料明细
    acc('1403.01', '种子', '借', 2, ['warehouse']),
    acc('1403.02', '饲料', '借', 2, ['warehouse']),
    acc('1403.03', '化肥', '借', 2, ['warehouse']),
    acc('1403.04', '农药', '借', 2, ['warehouse']),
    acc('1403.05', '其他材料', '借', 2, ['warehouse']),
    // 主营业务成本
    acc('5401.01', '农产品销售成本', '借', 2, ['department']),
    // 管理费用
    acc('5602.01', '管理人员薪酬', '借', 2, ['department']),
    acc('5602.02', '办公费', '借', 2, ['department']),
    acc('5602.03', '折旧费', '借', 2, ['department']),
    acc('5602.04', '差旅费', '借', 2, ['department']),
    acc('5602.05', '业务招待费', '借', 2, ['department']),
    acc('5602.06', '其他', '借', 2, ['department']),
  ];

  if (!isSB) {
    accts.push(acc('4301', '研发支出', '借', 1));
  }

  return accts;
}

// ============================================================
// 行业配置
// ============================================================
const INDUSTRIES = [
  { key: 'manufacturing', name: '制造业', fn: getManufacturingAccounts },
  { key: 'retail', name: '零售业', fn: getRetailAccounts },
  { key: 'service', name: '服务业', fn: getServiceAccounts },
  { key: 'construction', name: '建筑业', fn: getConstructionAccounts },
  { key: 'real_estate', name: '房地产业', fn: getRealEstateAccounts },
  { key: 'transport', name: '运输业', fn: getTransportAccounts },
  { key: 'agriculture', name: '农业', fn: getAgricultureAccounts },
];

const TAXPAYERS = [
  { key: 'general', name: '一般纳税人' },
  { key: 'small', name: '小规模纳税人' },
];

const STANDARDS = [
  { key: 'small_business', name: '小企业会计准则' },
  { key: 'enterprise', name: '企业会计准则' },
];

// ============================================================
// 生成单个模板
// ============================================================
function buildTemplate(standard, industry, taxpayer) {
  const stdInfo = STANDARDS.find(s => s.key === standard);
  const indInfo = INDUSTRIES.find(i => i.key === industry);
  const taxInfo = TAXPAYERS.find(t => t.key === taxpayer);

  const id = `${standard}_${industry}_${taxpayer}`;

  // 基础科目
  let accounts = getBaseAccounts(standard);

  // 企业准则特殊科目
  if (standard === 'enterprise') {
    accounts = accounts.concat(getEnterpriseOnlyAccounts());
  }

  // 应交税费明细
  if (taxpayer === 'general') {
    accounts = accounts.concat(getTaxDetailGeneral());
  } else {
    accounts = accounts.concat(getTaxDetailSmall());
  }

  // 行业特殊科目
  const indFn = indInfo.fn;
  accounts = accounts.concat(indFn(taxpayer, standard));

  // 去重（按code）
  const seen = new Set();
  accounts = accounts.filter(a => {
    if (seen.has(a.code)) return false;
    seen.add(a.code);
    return true;
  });

  return {
    id,
    name: `${indInfo.name}（${taxInfo.name}·${stdInfo.name}）`,
    standard: standard,
    industry: industry,
    taxpayer: taxpayer,
    version: '2026.1',
    accounts,
  };
}

// ============================================================
// 主流程
// ============================================================
function main() {
  const manifest = {
    version: '2026.1',
    standards: {
      small_business: { name: '小企业会计准则', description: '适用于小型企业' },
      enterprise: { name: '企业会计准则', description: '适用于上市公司、大型企业' },
    },
    industries: {
      manufacturing: { name: '制造业' },
      retail: { name: '零售业' },
      service: { name: '服务业' },
      construction: { name: '建筑业' },
      real_estate: { name: '房地产业' },
      transport: { name: '运输业' },
      agriculture: { name: '农业' },
    },
    taxpayer_types: {
      general: { name: '一般纳税人' },
      small: { name: '小规模纳税人' },
    },
    templates: [],
  };

  const templates = [];

  for (const std of STANDARDS) {
    const dir = path.join(BASE, std.key);
    fs.mkdirSync(dir, { recursive: true });

    // 写 base.json
    const baseAccts = getBaseAccounts(std.key);
    const baseTax = getTaxDetailGeneral().concat(getTaxDetailSmall());
    // base只包含通用科目+税费明细，不含行业特殊
    const baseObj = {
      id: `${std.key}_base`,
      name: `通用基础（${std.name}）`,
      standard: std.key,
      industry: 'general',
      taxpayer: 'general',
      version: '2026.1',
      accounts: baseAccts.concat(
        std.key === 'enterprise' ? getEnterpriseOnlyAccounts() : []
      ),
    };
    fs.writeFileSync(path.join(dir, 'base.json'), JSON.stringify(baseObj, null, 2), 'utf-8');

    for (const ind of INDUSTRIES) {
      for (const tax of TAXPAYERS) {
        const tpl = buildTemplate(std.key, ind.key, tax.key);
        const filename = `${ind.key}_${tax.key}.json`;
        fs.writeFileSync(path.join(dir, filename), JSON.stringify(tpl, null, 2), 'utf-8');

        manifest.templates.push({
          id: tpl.id,
          standard: std.key,
          industry: ind.key,
          taxpayer: tax.key,
          file: `${std.key}/${filename}`,
        });

        templates.push(tpl);
        console.log(`✅ ${tpl.id} (${tpl.accounts.length} accounts)`);
      }
    }
  }

  // 写 manifest
  fs.writeFileSync(path.join(BASE, 'manifest.json'), JSON.stringify(manifest, null, 2), 'utf-8');
  console.log(`\n📋 manifest.json written with ${manifest.templates.length} templates`);
  console.log(`📁 Total files: ${manifest.templates.length + STANDARDS.length * 1} (templates + base + manifest)`);
}

main();
