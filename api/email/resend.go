package email

import (
	"fmt"
	"os"
	"sync"

	"github.com/resend/resend-go/v2"
)

const baseStyle = `body { font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', sans-serif; background: #0a0a0a; margin: 0; padding: 40px 20px; }
.card { background: #111; border-radius: 16px; padding: 40px; max-width: 480px; margin: 0 auto; border: 1px solid rgba(255,255,255,0.06); }
.logo { font-size: 22px; font-weight: 700; color: #22c55e; margin-bottom: 28px; letter-spacing: -0.3px; }
h1 { color: #fafafa; font-size: 20px; margin: 0 0 16px; font-weight: 700; }
p { color: #a1a1aa; line-height: 1.6; margin: 0 0 20px; font-size: 14px; }
.btn { display: inline-block; background: #22c55e; color: #052e1a; padding: 12px 24px; border-radius: 10px; text-decoration: none; font-weight: 600; font-size: 14px; }
.pill { display: inline-block; background: rgba(34,197,94,0.12); color: #22c55e; padding: 6px 12px; border-radius: 999px; font-size: 13px; font-weight: 600; }
.footer { color: #52525b; font-size: 12px; margin-top: 28px; border-top: 1px solid rgba(255,255,255,0.05); padding-top: 20px; }`

var (
	once   sync.Once
	client *resend.Client
	from   string
)

func lazy() (*resend.Client, string, error) {
	once.Do(func() {
		key := os.Getenv("RESEND_API_KEY")
		from = os.Getenv("EMAIL_FROM")
		if from == "" {
			from = "rx-trace <onboarding@resend.dev>"
		}
		if key != "" {
			client = resend.NewClient(key)
		}
	})
	if client == nil {
		return nil, "", fmt.Errorf("RESEND_API_KEY not set")
	}
	return client, from, nil
}

func wrap(body string) string {
	return `<html><head><style>` + baseStyle + `</style></head><body><div class="card"><div class="logo">rx-trace</div>` +
		body +
		`<div class="footer">rx-trace, crowd-sourced medicine availability</div></div></body></html>`
}

type StockHit struct {
	Drug       string
	Pharmacy   string
	Address    string
	DistanceM  float64
	DirectionURL string
}

func SendStockAlert(to, drug string, hits []StockHit, unsubscribeURL string) error {
	c, fromAddr, err := lazy()
	if err != nil {
		return err
	}
	rows := ""
	for _, h := range hits {
		dist := ""
		if h.DistanceM > 0 {
			dist = fmt.Sprintf(" <span style='color:#71717a;font-size:12px;'>(%d m)</span>", int(h.DistanceM))
		}
		rows += fmt.Sprintf(`<div style="padding:12px 0;border-bottom:1px solid rgba(255,255,255,0.04);"><div style="font-weight:600;color:#fafafa;">%s%s</div><div style="color:#a1a1aa;font-size:13px;">%s</div></div>`,
			h.Pharmacy, dist, h.Address)
	}
	body := fmt.Sprintf(`<span class="pill">back in stock</span>
        <h1>%s is available near you</h1>
        <p>One or more pharmacies you asked us to watch just reported having it.</p>
        <div>%s</div>
        <p style="margin-top:24px;font-size:12px;color:#52525b;">
          Stock reports are crowd-sourced and can be stale. Call ahead if you can.
          <br><a href="%s" style="color:#71717a;">unsubscribe from this alert</a>
        </p>`, drug, rows, unsubscribeURL)
	_, err = c.Emails.Send(&resend.SendEmailRequest{
		From:    fromAddr,
		To:      []string{to},
		Subject: fmt.Sprintf("%s is in stock near you", drug),
		Html:    wrap(body),
	})
	return err
}
