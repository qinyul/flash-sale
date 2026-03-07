import http from "k6/http";
import { check, sleep } from "k6";
import { Counter } from "k6/metrics";

const TARGET =  __ENV.TARGET || 1000;

const successFullPurchase = new Counter("successful_purchases");
export const options = {
  stages: [
    { duration: "5s", target: TARGET }, // ramp up to TARGET users
    { duration: "10s", target: TARGET }, // stay at TARGET users
    { duration: "5s", target: 0 }, // ramp down to 0 users
  ],
};

const BASE_URL = __ENV.BASE_URL || "http://localhost:8080";

export function setup() {
  const params = { headers: { "Content-Type": "application/json" } };
  const productPayload = JSON.stringify({
    name: "Asics Gel-1130 Flash Sale",
    base_price: 1500000.0,
    quantity: 10,
  });

  const res = http.post(`${BASE_URL}/products`, productPayload, params);
  return { productId: res.json().data.product_id };
}

export default function (data) {
  const params = { headers: { "Content-Type": "application/json" } };

  // Everyone tries to buy the EXACT SAME PRODUCT
  const purchasePayload = JSON.stringify({
    product_id: data.productId,
    quantity: 1,
  });

  const res = http.post(`${BASE_URL}/orders`, purchasePayload, params);
  check(res, {
    "request processed safely (200 or 400)": (r) => {
      if (r.status === 200) {
        successFullPurchase.add(1);
      }
      return r.status === 200 || r.status === 400;
    },
    "too many request (429)": (r) => r.status === 429,
    "internal server error (500)": (r) => r.status === 500,
  });

  sleep(1);
}
