package proxy


import (
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/q5n/download-proxy/internal/config"
	"github.com/q5n/download-proxy/internal/security"
)



type Proxy struct {

	Config *config.Config


	Client *http.Client
}



func New(
	cfg *config.Config,
)*Proxy{


	client:=&http.Client{

		CheckRedirect: func(
			req *http.Request,
			via []*http.Request,
		) error {


			if len(via)>=10{
				return http.ErrUseLastResponse
			}

			return nil
		},
	}


	return &Proxy{

		Config:cfg,

		Client:client,
	}
}



func (p *Proxy) Handler(
	w http.ResponseWriter,
	r *http.Request,
){


	q:=r.URL.Query()


	target:=q.Get("url")


	expireStr:=q.Get("expire")

	sign:=q.Get("sign")



	if target=="" ||
		expireStr=="" ||
		sign=="" {

		http.Error(
			w,
			"missing parameter",
			400,
		)

		return
	}



	var expire int64

	_,err:=fmt.Sscan(
		expireStr,
		&expire,
	)

	if err!=nil{

		http.Error(
			w,
			"invalid expire",
			400,
		)

		return
	}



	if !security.Verify(
		target,
		expire,
		sign,
		p.Config.Secret,
		p.Config.MaxExpireSeconds,
	){

		http.Error(
			w,
			"invalid signature",
			403,
		)

		return
	}



	if !p.allowed(target){

		http.Error(
			w,
			"domain blocked",
			403,
		)

		return
	}



	req,err:=http.NewRequest(
		r.Method,
		target,
		nil,
	)


	if err!=nil{

		http.Error(
			w,
			err.Error(),
			500,
		)

		return
	}



	// 转发请求头
	req.Header=r.Header.Clone()



	resp,err:=p.Client.Do(req)


	if err!=nil{

		http.Error(
			w,
			err.Error(),
			502,
		)

		return
	}


	defer resp.Body.Close()



	copyHeader(
		w.Header(),
		resp.Header,
	)



	w.WriteHeader(
		resp.StatusCode,
	)



	// 流式复制
	io.Copy(
		w,
		resp.Body,
	)

}




func (p *Proxy) allowed(
	raw string,
) bool{


	u,err:=url.Parse(raw)


	if err!=nil{
		return false
	}



	host:=u.Hostname()



	for _,domain:=range p.Config.AllowedDomains{


		if host==domain ||
			strings.HasSuffix(
				host,
				"."+domain,
			){

			return true
		}

	}


	return false
}



func copyHeader(
	dst http.Header,
	src http.Header,
){

	for k,v:=range src{


		switch strings.ToLower(k){

		case "connection",
			"transfer-encoding",
			"keep-alive":
			continue

		}


		for _,vv:=range v{

			dst.Add(
				k,
				vv,
			)

		}

	}

}