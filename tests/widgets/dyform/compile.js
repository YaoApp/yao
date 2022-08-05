/**
 * The DSL compiler.
 * Translate the customize DSL to Models, Processes, Flows, Tables, etc.
 */

/**
 * Source
 * Where to get the source of DSL
 */
function Source() {
  return dyforms();
}

/**
 * Compile
 * Translate or extend the customize DSL
 * @param {*} dsl
 */
function Compile(name, dsl) {
  let newdsl = { name: name };
  return newdsl;
}

/**
 * OnLoad
 * When the widget instance are loaded, the function will be called.
 * For preparing the sources the widget need.
 * @param {DSL} dsl
 */
function OnLoad(name, dsl) {
  console.log(name, dsl);
}

/**
 * Migrate
 * When the migrate command executes, the function will be called
 * @param {DSL} dsl
 * @param {bool} force
 */
function Migrate(dsl, force) {}

/**
 * Customize DSL
 * @returns
 */
function dyforms() {
  return {
    pad: {
      guard: "bearer-jwt",
      title: "应用表单",
      sn: "PAD1024",
      orign: "手机平板",
      columns: [
        {
          name: "门店",
          field: "store_id",
          searchable: true,
          props: {
            value: ":store_id",
            type: "SelectStore",
            title: "门店",
            placeholder: "请选择门店",
            validation: [
              { type: "Required" },
              { type: "StoreStatus", args: [":store_id"] },
            ],
          },
        },
        {
          name: "订单数量",
          field: "orders_amount",
          searchable: true,
          props: {
            value: ":orders_amount",
            type: "Number",
            title: "订单数量",
            placeholder: "今日订单数量",
            computed: "OrderAmount",
            validation: [
              { type: "Required" },
              { type: "IsNumber", args: [":orders_amount"] },
            ],
          },
        },
        {
          name: "账单地址",
          field: "address",
          props: {
            value: ":address",
            type: "Text",
            title: "账单地址",
            placeholder: "填写账单地址",
            validation: [
              { type: "Required" },
              { type: "MaxLength", args: [":address", 200] },
            ],
          },
        },
      ],
      filters: [],
      "list-layout": {
        columns: [{ name: "store_id" }, { name: "orders_amount" }],
        filters: [{ name: "订单数量" }],
      },
    },
  };
}
