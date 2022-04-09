package handler

import (
	"fmt"
	"log"
	"sync"

	"github.com/artnoi43/stubborn/lib/dnsutils"
	"github.com/miekg/dns"
)

func (h *handler) HandleWithDoT(w dns.ResponseWriter, r *dns.Msg) {
	msg := new(dns.Msg).SetReply(r)
	msg.Compress = false
	msg.RecursionDesired = true
	msg.RecursionAvailable = true

	ip := h.conf.DoT.UpStreamIp
	port := h.conf.DoT.UpStreamPort

	var cacheMissed bool
	var m = make(answerMap)
	var wg sync.WaitGroup
	for _, q := range r.Question {
		tString := dnsutils.DnsTypes[q.Qtype]
		k := dnsutils.NewKey(q.Name, q.Qtype)
		cached, found := h.repository.Get(k)
		if found {
			log.Printf("cache hit for \"%s\" [%s]", k.String(), tString)
			msg.Answer = append(msg.Answer, cached...)
		} else {
			log.Printf("cache missed for \"%s\" [%s]", k.String(), tString)
			cacheMissed = true
			ans, rtt, err := h.dotClient.Exchange(r, fmt.Sprintf("%v:%v", ip, port))
			if err != nil {
				log.Printf("failed to connect upstream: %v\n", err.Error())
				return
			}
			log.Println(ans, rtt)
			// log.Printf("answer for '%s' received in %s", q.String(), rtt.String())
			if ans.Answer != nil {
				msg.Answer = append(msg.Answer, ans.Answer...)
				m[k] = append(m[k], ans.Answer...)
			}
		}
	}
	w.WriteMsg(msg)
	if cacheMissed && len(m) > 0 {
		h.repository.SetMap(m)
	}
	wg.Wait()
}