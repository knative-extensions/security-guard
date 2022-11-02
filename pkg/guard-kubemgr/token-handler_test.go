/*
Copyright 2022 The Knative Authors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package guardkubemgr

import (
	"testing"

	authv1 "k8s.io/api/authentication/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"

	k8sfake "k8s.io/client-go/kubernetes/fake"
	ctest "k8s.io/client-go/testing"
)

const testToken = "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCIsImtpZCI6IjEyMzQ1NiJ9.eyJpc3MiOiJodHRwczovL3h4eHh4LmF1dGgwLmNvbS8iLCJzdWIiOiJhdXRoMHwxMjM0NTY3ODkiLCJhdWQiOiJzZWN1cml0eS1ndWFyZCIsImlhdCI6MTYzNDMzMjg5NSwiZXhwIjoxNjM0NDE5Mjk1LCJhenAiOiJNWV9DTElFTlRfSURfMTIzNDU2Iiwic2NvcGUiOiJvcGVuaWQgZW1haWwiLCJwZXJtaXNzaW9ucyI6W119.RnoWlTU2UwfllQv4AwmUNa6kVISJdQrfJLRt1oW_c_A"

func TestKubeMgr_TokenData(t *testing.T) {
	k := new(KubeMgr)
	k.getConfigFunc = fakeGetInclusterConfig

	tokenReview := &authv1.TokenReview{
		TypeMeta: metav1.TypeMeta{
			Kind:       "TokenReview",
			APIVersion: "authentication.k8s.io/v1",
		},
		Status: authv1.TokenReviewStatus{
			Authenticated: true,
			Audiences:     []string{ServiceAudience},
			User:          authv1.UserInfo{},
		},
	}

	t.Run("base", func(t *testing.T) {
		cmClient := k8sfake.Clientset{}
		cmClient.AddReactor("create", "tokenreviews", func(action ctest.Action) (handled bool, ret runtime.Object, err error) {
			return true, tokenReview, nil
		})
		k.cmClient = &cmClient

		sid, ns, err := k.TokenData(testToken)
		if err == nil {
			// TBD investigate fake client behavior
			t.Errorf("fake client always produce an error %s", err.Error())
			return

		}
		if ns != "" {
			t.Errorf("KubeMgr.TokenData() ns = %v", ns)
			return
		}
		if sid != "" {
			t.Errorf("KubeMgr.TokenData() sid = %v", sid)
			return
		}
	})

}

func TestKubeMgr_parseJwt(t *testing.T) {

	tests := []struct {
		name        string
		token       string
		wantPodname string
		wantNs      string
		wantErr     bool
	}{
		{
			name:    "noToken",
			token:   "",
			wantErr: true,
		},
		{
			name:    "badToken",
			token:   "abc",
			wantErr: true,
		},
		{
			name:        "goodToken",
			token:       "eyJhbGciOiJSUzI1NiIsImtpZCI6InNGX0kyeTBRQlFqbWRJMnltbjlDbi1TX0RFZ1ZlVXRTTFR1Z1lCSkEzQ1UifQ.eyJhdWQiOlsiZ3VhcmQtc2VydmljZSJdLCJleHAiOjE2NjU3NTUzNDIsImlhdCI6MTY2NTc0ODE0MiwiaXNzIjoiaHR0cHM6Ly9rdWJlcm5ldGVzLmRlZmF1bHQuc3ZjIiwia3ViZXJuZXRlcy5pbyI6eyJuYW1lc3BhY2UiOiJkZWZhdWx0IiwicG9kIjp7Im5hbWUiOiJoZWxsb3dvcmxkLWdvLTAwMDAxLWRlcGxveW1lbnQtNWI5ZjlkZDVmYi1ud3FqbCIsInVpZCI6IjM5MmNiOGNhLWRlYzctNDE3MS05NzFkLWIxYWYwZTA2ZmVlZCJ9LCJzZXJ2aWNlYWNjb3VudCI6eyJuYW1lIjoiZGVmYXVsdCIsInVpZCI6ImQ3NjUwNmNmLTA1NGYtNDI0MS05ZDU4LWNlOWFkM2Y5NTZmMCJ9fSwibmJmIjoxNjY1NzQ4MTQyLCJzdWIiOiJzeXN0ZW06c2VydmljZWFjY291bnQ6ZGVmYXVsdDpkZWZhdWx0In0.Xph8d8yq1wHYsCZL6Aw7jIDDzVrt-z3uNFqrbJuiYRgsDXjbnvLv-UsyWGCelW_7QTroYVQwCKYUap61QVbYZ5Q4kgVY-Kf-pjvKg3PXZgsluxa6FlIm4BDX4cpYz1lekLNjpI5TPijcRb1Ka-z9wJWboudLVsKb_KFw5iUnQb68eDiEjRkoyF5ZhNCJnU89wTvhyadw1FyghB0s3AQuyLXZkaRlseqUK8a0G8dFrR6nau2OomODi1VMZwn5T1voQlo5KOhmWgeh23WlXv6g_yp9oSG3kgpnpY8dMAasPpSAEl2XRC8p--GDuUC0R_tx6-QuRKnNzg5UiaQF92nIDhn6zW7cQYl4yOE0iSRMG2HnstWRYBaQQ6k1FOH8QEPUoUCrwrd9ZeU1uTIMJ6ssOYCqaRFbHAPqXOu7nV0MOvKOhnDONQwCQ0zcK0fGqUEOuZ31YLcHgNzL6YX7km1f4fLaIII-PMTugMlQzpbPoIiPAc6Bfix4ei7unHDfNbRz5TG87Y3RJ_OQSG4gCpk_fqRwG549PqY-NmCeF_Il7WkDm7us_PEh9zcx9du3EJouDU05u5OMbpCZ6LLCLgGu1bjngpLZYLujj1cRfXpqGY-pQFA_4QMkl7Z7XRBcxr7QETQZ1g5qj0psh4s0203A5plP3qdwtu1BNCNZPaKvSLE",
			wantNs:      "default",
			wantPodname: "helloworld-go-00001-deployment-5b9f9dd5fb-nwqjl",
			wantErr:     false,
		},
		{
			name:        "noNamespace",
			token:       "eyJhbGciOiJSUzI1NiIsImtpZCI6InNGX0kyeTBRQlFqbWRJMnltbjlDbi1TX0RFZ1ZlVXRTTFR1Z1lCSkEzQ1UifQ.eyJhdWQiOlsiZ3VhcmQtc2VydmljZSJdLCJleHAiOjE2NjU3NTUzNDIsImlhdCI6MTY2NTc0ODE0MiwiaXNzIjoiaHR0cHM6Ly9rdWJlcm5ldGVzLmRlZmF1bHQuc3ZjIiwia3ViZXJuZXRlcy5pbyI6eyJuYW1lc3BhY2UiOiJkZWZhdWx0IiwicG9kIjp7Im5hbWUiOiJoZWxsb3dvcmxkLWdvLTAwMDAxLWRlcGxveW1lbnQtNWI5ZjlkZDVmYi1ud3FqbCIsInVpZCI6IjM5MmNiOGNhLWRlYzctNDE3MS05NzFkLWIxYWYwZTA2ZmVlZCJ9LCJzZXJ2aWNlYWNjb3VudCI6eyJuYW1lIjoiZGVmYXVsdCIsInVpZCI6ImQ3NjUwNmNmLTA1NGYtNDI0MS05ZDU4LWNlOWFkM2Y5NTZmMCJ9fSwibmJmIjoxNjY1NzQ4MTQyLCJzdWIiOiJzeXN0ZW06c2VydmljZWFjY291bnQ6ZGVmYXVsdDpkZWZhdWx0In0.gw6_H80sABvXazaI9PxG3NnHiKeq4VCGbEroLCuWb0R6EHZ6kgIhC8bZROiWeNBaFLPqvbZmR0TxuQn3jjqPlAdl8Dxx00zUiTwpXoyPppnmcpQNbs7Wky7RYPFJ83dEowjvKnABrA-kJeqHOR7RigDvK0pV6AE8rrw9JdlZynl1U8UqA9s_M9zz1c2N8w-erkEwWM93xjda7z2Y4hjFmxEoHr7JzY8iLm9RNO4PLNPWo_Vqh37XG_j_rBeXIG5n1RC7G-3EGix-UiYCePrTn2dAHQOkCNSsR1qF4Dvg6-fX7DrxmlEwsUdqXsADQEp2JsFkkRtJ4XfUF-3y4_XKmg",
			wantNs:      "default",
			wantPodname: "helloworld-go-00001-deployment-5b9f9dd5fb-nwqjl",
			wantErr:     false,
		},
		{
			name:    "noKubernetesIo",
			token:   "eyJhbGciOiJSUzI1NiIsImtpZCI6InNGX0kyeTBRQlFqbWRJMnltbjlDbi1TX0RFZ1ZlVXRTTFR1Z1lCSkEzQ1UifQ.eyJhdWQiOlsiZ3VhcmQtc2VydmljZSJdLCJleHAiOjE2NjU3NTUzNDIsImlhdCI6MTY2NTc0ODE0MiwiaXNzIjoiaHR0cHM6Ly9rdWJlcm5ldGVzLmRlZmF1bHQuc3ZjIiwibmJmIjoxNjY1NzQ4MTQyLCJzdWIiOiJzeXN0ZW06c2VydmljZWFjY291bnQ6ZGVmYXVsdDpkZWZhdWx0In0.m-kOXmvMimvOe_PzqAn3m4OYP7ULN_KECfwJzvfg7QzhpVDtONxJOsyuz84QJ5xKyHlv_duOUp8bUsHm2XGTUSF9aWewvzu27Qg--L-OeuoZwcH1vySCVhdgPaDENghRfUYLSLekJZ0iKsqWSuxZ0l6OvyLOojZ-jYCihK2jKP4mWxeJqwkSpZsZj0cwsVp9I9y53-h56duttFhbDAGFB5-oLIhZumdVp9nvJYpflgu943q-IizgpBvIrxydovDzXH81e-j0sTui0h54IehjUIOFVpm7p_x0m5z37V1yNTrv2Kd2pXd1UhaEOM0y6_PFE65YpP8EUAQjEgciRo1Z9A",
			wantErr: true,
		},
		{
			name:    "noNamespace",
			token:   "eyJhbGciOiJSUzI1NiIsImtpZCI6InNGX0kyeTBRQlFqbWRJMnltbjlDbi1TX0RFZ1ZlVXRTTFR1Z1lCSkEzQ1UifQ.eyJhdWQiOlsiZ3VhcmQtc2VydmljZSJdLCJleHAiOjE2NjU3NTUzNDIsImlhdCI6MTY2NTc0ODE0MiwiaXNzIjoiaHR0cHM6Ly9rdWJlcm5ldGVzLmRlZmF1bHQuc3ZjIiwia3ViZXJuZXRlcy5pbyI6eyJwb2QiOnsibmFtZSI6ImhlbGxvd29ybGQtZ28tMDAwMDEtZGVwbG95bWVudC01YjlmOWRkNWZiLW53cWpsIiwidWlkIjoiMzkyY2I4Y2EtZGVjNy00MTcxLTk3MWQtYjFhZjBlMDZmZWVkIn0sInNlcnZpY2VhY2NvdW50Ijp7Im5hbWUiOiJkZWZhdWx0IiwidWlkIjoiZDc2NTA2Y2YtMDU0Zi00MjQxLTlkNTgtY2U5YWQzZjk1NmYwIn19LCJuYmYiOjE2NjU3NDgxNDIsInN1YiI6InN5c3RlbTpzZXJ2aWNlYWNjb3VudDpkZWZhdWx0OmRlZmF1bHQifQ.Co1QfrHBwQDc8aJg2dt-39MrJNYq7JRo86rttt3Xpxr4UBxggTYqRhxtvpk5sVPJuz4FA__gY6F0ZkbScVPWVATLBctNZ0lOr-PzX_yZkp0H0jFIxpVivdyqzws0md2v6_hNIq9Z8Tct8mvh7DcL9jYm7lNuKT_yh_hl1BEZs99HodyFMdbemLGjiwFH5E7aAF7TIDWbqnaYtkrHg_CnudotGq98JvqAVV8I8hc9asjy900HHspNi0gf5FvGxr7wULEvsOWWkJVP7sz7T3BgWsw2uPIhnrwZgXfj2GxRfG3IaNm7CTEwYdUk0cVIPVvxUGciH6ys7W-pORLxMFb7Uw",
			wantErr: true,
		},
		{
			name:    "emptyNamespace",
			token:   "eyJhbGciOiJSUzI1NiIsImtpZCI6InNGX0kyeTBRQlFqbWRJMnltbjlDbi1TX0RFZ1ZlVXRTTFR1Z1lCSkEzQ1UifQ.eyJhdWQiOlsiZ3VhcmQtc2VydmljZSJdLCJleHAiOjE2NjU3NTUzNDIsImlhdCI6MTY2NTc0ODE0MiwiaXNzIjoiaHR0cHM6Ly9rdWJlcm5ldGVzLmRlZmF1bHQuc3ZjIiwia3ViZXJuZXRlcy5pbyI6eyJuYW1lc3BhY2UiOiIiLCJwb2QiOnsibmFtZSI6IiIsInVpZCI6IjM5MmNiOGNhLWRlYzctNDE3MS05NzFkLWIxYWYwZTA2ZmVlZCJ9LCJzZXJ2aWNlYWNjb3VudCI6eyJuYW1lIjoiZGVmYXVsdCIsInVpZCI6ImQ3NjUwNmNmLTA1NGYtNDI0MS05ZDU4LWNlOWFkM2Y5NTZmMCJ9fSwibmJmIjoxNjY1NzQ4MTQyLCJzdWIiOiJzeXN0ZW06c2VydmljZWFjY291bnQ6ZGVmYXVsdDpkZWZhdWx0In0.nPDs212znXDveKnPny9QKAJpwZxsNm-_897UwNi1wUjT254hlB9QiOU1bZSpWZF4nZnomcXadTTf6zKCxOq0YQy7ulPEzUd9SBVRcO8Io_huOHNm5YVtSyesnUdF-R4DNop_RlLL29iqLLCX6P89oK_MvYiwPpjSe3-mlT763ktC0FbHXHXyxQXbdA96PXReYtpFfkGqPruLopWH-7WEsjdQq-p8R9rvqLvZIf54yvfGtw2w7JXXMXeuWahrwpgzK6PfF9mdj2C4g2Bk355vL9NBLfGJs6xG5cu69Sz5Jyl4T9rTRA-YyEsozZ5stMFwRBCCKWlbFthg27_bV28qGw",
			wantErr: true,
		},
		{
			name:    "noPodname",
			token:   "eyJhbGciOiJSUzI1NiIsImtpZCI6InNGX0kyeTBRQlFqbWRJMnltbjlDbi1TX0RFZ1ZlVXRTTFR1Z1lCSkEzQ1UifQ.eyJhdWQiOlsiZ3VhcmQtc2VydmljZSJdLCJleHAiOjE2NjU3NTUzNDIsImlhdCI6MTY2NTc0ODE0MiwiaXNzIjoiaHR0cHM6Ly9rdWJlcm5ldGVzLmRlZmF1bHQuc3ZjIiwia3ViZXJuZXRlcy5pbyI6eyJuYW1lc3BhY2UiOiJkZWZhdWx0IiwicG9kIjp7InVpZCI6IjM5MmNiOGNhLWRlYzctNDE3MS05NzFkLWIxYWYwZTA2ZmVlZCJ9LCJzZXJ2aWNlYWNjb3VudCI6eyJuYW1lIjoiZGVmYXVsdCIsInVpZCI6ImQ3NjUwNmNmLTA1NGYtNDI0MS05ZDU4LWNlOWFkM2Y5NTZmMCJ9fSwibmJmIjoxNjY1NzQ4MTQyLCJzdWIiOiJzeXN0ZW06c2VydmljZWFjY291bnQ6ZGVmYXVsdDpkZWZhdWx0In0.BXPkIGzit1MpXkosKR_JLMgNhDEOFa8M-zukljjxoZFKDEk61FwxQ-QhX3RKDLFjRXNKj4MrxBAV0Rbzac_Epv-A_KE9YJYA7KvImzqap-cTqSBlboOiiW7FcxJdziM8XAw19ci-mC12Dq29RtXPLvD5BTAsSqsmb2D31SuChzrqYBr1igAwN4tDi13QllMJYlIORZ_t2A0SA9rBc3aev_SWthVL0LJTvP4eq2m06f6E8YILYui1-NzB4FZk7dIOOut8UKP2zRqu9VB73EA0ya2GlzhF-rwQA8PJBe-3PXhAJgr0lnm2PfWLFbhUp8Mgi9LYoMowBgmwbSkfVfaiVw",
			wantNs:  "default",
			wantErr: true,
		},
		{
			name:    "emptyPodname",
			token:   "eyJhbGciOiJSUzI1NiIsImtpZCI6InNGX0kyeTBRQlFqbWRJMnltbjlDbi1TX0RFZ1ZlVXRTTFR1Z1lCSkEzQ1UifQ.eyJhdWQiOlsiZ3VhcmQtc2VydmljZSJdLCJleHAiOjE2NjU3NTUzNDIsImlhdCI6MTY2NTc0ODE0MiwiaXNzIjoiaHR0cHM6Ly9rdWJlcm5ldGVzLmRlZmF1bHQuc3ZjIiwia3ViZXJuZXRlcy5pbyI6eyJuYW1lc3BhY2UiOiJkZWZhdWx0IiwicG9kIjp7Im5hbWUiOiIiLCJ1aWQiOiIzOTJjYjhjYS1kZWM3LTQxNzEtOTcxZC1iMWFmMGUwNmZlZWQifSwic2VydmljZWFjY291bnQiOnsibmFtZSI6ImRlZmF1bHQiLCJ1aWQiOiJkNzY1MDZjZi0wNTRmLTQyNDEtOWQ1OC1jZTlhZDNmOTU2ZjAifX0sIm5iZiI6MTY2NTc0ODE0Miwic3ViIjoic3lzdGVtOnNlcnZpY2VhY2NvdW50OmRlZmF1bHQ6ZGVmYXVsdCJ9.UT9Bbzv5vF5JX7hzMAcEk8EPcmHnqAcHDIHOoYfRCWNglDFRQg5u4ghA8ptIF0laH-sfIJoRWIShsFVS3VS08hVNSr_gpvkIXHCRZKSNr_Bzx0DQo1xKY9540swq4_-qCLU_qMcn-H7IFVL0wystuiTqj2ps_N25Q-NUAGOJyKCchc8nxlEViUjFVW7LPhFKLbtj6dWxMtg6Sx9VBeFeYnbrRxppcB_LVCAbNus-4iIBzodR2BqoRpe9ou6A7pNRrD1MlEn7r6z4Wsrw0Y_5-nhMYP4MfNNvGQYIs0Flt1aVGWmmMOb5ojtJZouy3E1g2icGIb8gZT7dvWHCitkuow",
			wantNs:  "default",
			wantErr: true,
		},
		{
			name:    "noPod",
			token:   "eyJhbGciOiJSUzI1NiIsImtpZCI6InNGX0kyeTBRQlFqbWRJMnltbjlDbi1TX0RFZ1ZlVXRTTFR1Z1lCSkEzQ1UifQ.eyJhdWQiOlsiZ3VhcmQtc2VydmljZSJdLCJleHAiOjE2NjU3NTUzNDIsImlhdCI6MTY2NTc0ODE0MiwiaXNzIjoiaHR0cHM6Ly9rdWJlcm5ldGVzLmRlZmF1bHQuc3ZjIiwia3ViZXJuZXRlcy5pbyI6eyJuYW1lc3BhY2UiOiJkZWZhdWx0Iiwic2VydmljZWFjY291bnQiOnsibmFtZSI6ImRlZmF1bHQiLCJ1aWQiOiJkNzY1MDZjZi0wNTRmLTQyNDEtOWQ1OC1jZTlhZDNmOTU2ZjAifX0sIm5iZiI6MTY2NTc0ODE0Miwic3ViIjoic3lzdGVtOnNlcnZpY2VhY2NvdW50OmRlZmF1bHQ6ZGVmYXVsdCJ9.CZ0ScFlPUISc2PWqKJM0Z0pSrQHAY_urJcTePltlZshofKiYC3f4lFJxn3XwfGMpSW55qmGXph2oOk6P9gT_OD-gTeOiMMsL9ATjxMmyfPzcxhyUt59u2-ehCFrEiHNKtUaQkaOEW9CQ5lJJWt2ld64T9UaKW94-NoI0OPKDbmut84vc9VTXZiC4mcGVIxoDI7qtS_uqFGgttAztc5H7TSInaouiqsU_eDVeuVovZ-3QuymXPTdMQDiwjLt_g_1Z9gwOGeIfAdWcTh6xb_hLVxjOGciuFb0K-40hjWnr9LHZOfOABdxjKzrWN4nqH2owezn1lx9R-4d8O2P3pLGUGQ",
			wantNs:  "default",
			wantErr: true,
		},
		{
			name:    "wrongClaims",
			token:   "eyJhbGciOiJSUzI1NiIsImtpZCI6InNGX0kyeTBRQlFqbWRJMnltbjlDbi1TX0RFZ1ZlVXRTTFR1Z1lCSkEzQ1UifQ.ImFiYyI.g_IlGHhpGVGzsklChRGrsYzU8xlJlo6nRLV1KZGm_appyCsENw00_ZDQkduiAB1zGWzTPcRM4soSdvCPFvj7ymHsOo6F6vBInvG70KjqrSHlFmvaJBLsZCyFqGf3I8gzvHee2rpp9LNYKsZ-C6zXMEf3wGgwe9snZCTRWjyc8J21vw5p13E1WEd-SP0Tk4ry6LRPeX2ZTr4Qgbpa9rjcDHJd4f8zC5mimDpG0QQfYSerN-5FWGp2t6Bglb0jo8yYpblirnEByRsPr-mySnVfOjc_tDON46XIgls2UCaZ01s4EALHIElRBM5-CQJ-HP3XGucnASus4Nliay-dv6xMZg",
			wantErr: true,
		},
	}
	for _, tt := range tests {

		t.Run(tt.name, func(t *testing.T) {
			k := new(KubeMgr)
			k.getConfigFunc = fakeGetInclusterConfig
			k.cmClient = k8sfake.NewSimpleClientset(&v1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:        "guardian.sid",
					Namespace:   "ns",
					Annotations: map[string]string{},
				},
				Data: map[string]string{"Guardian": "{\"control\": {}}"},
			})
			gotPodname, gotNs, err := k.parseJwt(tt.token)
			if (err != nil) != tt.wantErr {
				t.Errorf("KubeMgr.parseJwt() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if gotPodname != tt.wantPodname {
				t.Errorf("KubeMgr.parseJwt() gotPodname = %v, want %v", gotPodname, tt.wantPodname)
			}
			if gotNs != tt.wantNs {
				t.Errorf("KubeMgr.parseJwt() gotNs = %v, want %v", gotNs, tt.wantNs)
			}
		})
	}
}

func TestKubeMgr_getPodData(t *testing.T) {
	tests := []struct {
		labels  map[string]string
		name    string
		podname string
		ns      string
		wantSid string
		wantErr bool
	}{
		{
			name:    "app",
			labels:  map[string]string{"app": "mysid"},
			podname: "mypod",
			ns:      "myns",
			wantSid: "mysid",
			wantErr: false,
		},
		{
			name:    "service",
			labels:  map[string]string{"serving.knative.dev/service": "mysid"},
			podname: "mypod",
			ns:      "myns",
			wantSid: "mysid",
			wantErr: false,
		},
		{
			name:    "no label",
			labels:  map[string]string{},
			podname: "mypod",
			ns:      "myns",
			wantSid: "mypod",
			wantErr: false,
		},
		{
			name:    "no such pod",
			labels:  map[string]string{},
			podname: "xxx",
			ns:      "myns",
			wantSid: "",
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			k := new(KubeMgr)
			k.getConfigFunc = fakeGetInclusterConfig
			k.cmClient = k8sfake.NewSimpleClientset(&v1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:        "mypod",
					Namespace:   "myns",
					Annotations: map[string]string{},
					Labels:      tt.labels,
				},
			})
			gotSid, err := k.getPodData(tt.podname, tt.ns, nil)
			if (err != nil) != tt.wantErr {
				t.Errorf("KubeMgr.getPodData() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if gotSid != tt.wantSid {
				t.Errorf("KubeMgr.getPodData() = %v, want %v", gotSid, tt.wantSid)
			}
		})
	}
}
