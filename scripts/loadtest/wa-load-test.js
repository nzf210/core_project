import http from 'k6/http';
import { check, sleep } from 'k6';
import { Rate, Trend, Counter } from 'k6/metrics';

// Custom metrics
const errorRate = new Rate('errors');
const messageDuration = new Trend('message_duration');
const rateLimitErrors = new Counter('rate_limit_errors');
const connectionErrors = new Counter('connection_errors');

// Test configuration
export const options = {
  scenarios: {
    // Scenario 1: Gradual ramp-up (stress test)
    stress_test: {
      executor: 'ramping-vus',
      startVUs: 0,
      stages: [
        { duration: '2m', target: 10 },  // Ramp up to 10 tenants
        { duration: '5m', target: 10 },  // Stay at 10 tenants
        { duration: '2m', target: 20 },  // Ramp up to 20 tenants
        { duration: '5m', target: 20 },  // Stay at 20 tenants
        { duration: '2m', target: 0 },   // Ramp down
      ],
      gracefulRampDown: '30s',
    },

    // Scenario 2: Spike test
    spike_test: {
      executor: 'ramping-vus',
      startTime: '20m',
      startVUs: 5,
      stages: [
        { duration: '30s', target: 5 },
        { duration: '1m', target: 50 },  // Sudden spike
        { duration: '2m', target: 50 },  // Hold spike
        { duration: '30s', target: 5 },  // Drop back
      ],
      gracefulRampDown: '30s',
    },
  },

  thresholds: {
    'http_req_duration': ['p(95)<5000'], // 95% of requests under 5s
    'errors': ['rate<0.1'],               // Error rate under 10%
    'http_req_failed': ['rate<0.1'],      // Failed requests under 10%
  },
};

const BASE_URL = __ENV.BASE_URL || 'http://localhost:8202';

export default function () {
  const tenantId = `test-tenant-${__VU}`;
  const target = '628123456789@s.whatsapp.net';
  const message = `Load test message from VU ${__VU} at ${Date.now()}`;

  const params = {
    headers: {
      'Content-Type': 'application/x-www-form-urlencoded',
    },
    tags: {
      tenant: tenantId,
    },
  };

  const payload = `tenant_id=${tenantId}&target=${target}&message=${encodeURIComponent(message)}`;

  const response = http.post(`${BASE_URL}/api/wa/send`, payload, params);

  // Record metrics
  messageDuration.add(response.timings.duration);

  // Check response
  const success = check(response, {
    'status is 200': (r) => r.status === 200,
    'response has success': (r) => {
      try {
        const body = JSON.parse(r.body);
        return body.success === true;
      } catch (e) {
        return false;
      }
    },
  });

  if (!success) {
    errorRate.add(1);

    // Track specific error types
    if (response.status === 429) {
      rateLimitErrors.add(1);
    } else if (response.status === 401 || response.status === 503) {
      connectionErrors.add(1);
    }

    console.log(`Error for tenant ${tenantId}: ${response.status} - ${response.body}`);
  } else {
    errorRate.add(0);
  }

  // Realistic think time between messages (1-3 seconds)
  sleep(1 + Math.random() * 2);
}

export function handleSummary(data) {
  return {
    'summary.json': JSON.stringify(data),
    stdout: textSummary(data, { indent: ' ', enableColors: true }),
  };
}

function textSummary(data, options) {
  const indent = options.indent || '';
  const enableColors = options.enableColors || false;

  let summary = '\n' + indent + '=== Load Test Summary ===\n\n';

  // HTTP metrics
  summary += indent + 'HTTP Requests:\n';
  summary += indent + `  Total: ${data.metrics.http_reqs.values.count}\n`;
  summary += indent + `  Failed: ${data.metrics.http_req_failed.values.passes} (${(data.metrics.http_req_failed.values.rate * 100).toFixed(2)}%)\n`;
  summary += indent + `  Duration (p95): ${data.metrics.http_req_duration.values['p(95)'].toFixed(2)}ms\n`;
  summary += indent + `  Duration (p99): ${data.metrics.http_req_duration.values['p(99)'].toFixed(2)}ms\n\n`;

  // Custom metrics
  summary += indent + 'WA Gateway Metrics:\n';
  summary += indent + `  Error Rate: ${(data.metrics.errors.values.rate * 100).toFixed(2)}%\n`;
  summary += indent + `  Rate Limit Errors: ${data.metrics.rate_limit_errors.values.count}\n`;
  summary += indent + `  Connection Errors: ${data.metrics.connection_errors.values.count}\n`;
  summary += indent + `  Message Duration (avg): ${data.metrics.message_duration.values.avg.toFixed(2)}ms\n`;
  summary += indent + `  Message Duration (p95): ${data.metrics.message_duration.values['p(95)'].toFixed(2)}ms\n\n`;

  // VUs
  summary += indent + 'Virtual Users:\n';
  summary += indent + `  Max: ${data.metrics.vus_max.values.max}\n`;
  summary += indent + `  Concurrent (avg): ${data.metrics.vus.values.value.toFixed(2)}\n\n`;

  return summary;
}
