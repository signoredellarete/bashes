# Code Review - Bashes

Data: 2026-07-11
Ambito: backend Go/Wails, datastore JSON, SSH/tunnel, shell locale, file transfer, frontend, test, workflow e packaging.

Revisione statica dello stato corrente. Il report precedente e' archiviato in `CODE_REVIEW.md.OLD`; i problemi gia' risolti non sono ripetuti.

## Riepilogo

| # | Problema | Area | Priorita' | Stato fix |
|---|---|---|---|---|
| 1 | Il bypass della verifica host key viene salvato e riutilizzato | Sicurezza SSH | Alta | Fatto |
| 2 | Copie e upload sovrascrivono direttamente la destinazione e lasciano file parziali | File transfer | Alta | Fatto |
| 3 | Un errore UI dopo l'avvio della shell viene riportato come errore di connessione e puo' lasciare una sessione orfana | Frontend/sessioni | Alta | Fatto |
| 4 | I job di trasferimento non vengono rimossi e l'avvio concorrente non e' atomico | Backend/runtime | Media | Fatto |
| 5 | I confini del file manager sono aggirabili con symlink; il move di path assoluti e' troppo permissivo | Sicurezza file | Media | Fatto |
| 6 | Modificare endpoint SSH conserva la fingerprint del vecchio server | Sicurezza/UX | Media | Fatto |
| 7 | La consistenza tra nodi salvati e risorse runtime dipende dal frontend | Design backend | Media | Fatto |
| 8 | Un tunnel SOCKS puo' essere esposto senza autenticazione e senza timeout sulle connessioni inattive | Sicurezza tunnel | Media | Fatto |
| 9 | `main.js` e `app.go` concentrano troppe responsabilita'; il fallback demo nasconde errori di binding | Manutenibilita' | Media | Fatto |
| 10 | Non esistono test frontend ne' test dei cicli di vita runtime | Qualita' | Media | Fatto |
| 11 | Autenticazione, errori ed eventi sono duplicati o codificati come stringhe | Design/API | Media | Fatto |
| 12 | Il workflow CI assegna permessi di scrittura globali e le release non pubblicano checksum | Supply chain | Media | Fatto |
| 13 | Scrittura settings e generazione chiavi non sono transazioni atomiche complete | Persistenza | Bassa | Fatto |
| 14 | Workflow e versioni sono duplicati in molti punti | Manutenibilita' CI | Bassa | Fatto |

## Problemi e correzioni

### 1. Bypass host key persistente

`authPreferenceFromSessionInput` salva `TrustHostKey` e `applyAuthPreference` lo riattiva nelle connessioni successive (`app.go:1735-1774`). Il testo UI parla invece di bypass "for this session". Una singola scelta insicura disabilita quindi TOFU/`known_hosts` in modo permanente e invisibile.

**Correzione:** non persistere mai il bypass. Conservare solo metodo e riferimento alla chiave; usare `TrustHostKey` esclusivamente nella richiesta corrente. Migrare i record esistenti ignorando o rimuovendo `auth.trustHostKey`.

### 2. Trasferimenti non atomici e sovrascrittura silenziosa

`openWriter` usa `os.Create`/`sftp.Create` (`file_transfer.go:1166-1187`): tronca subito un file esistente. Su errore, annullamento o caduta SSH resta una destinazione parziale; un move puo' poi produrre risultati ambigui. Non esiste una policy esplicita per i conflitti.

**Correzione:** scrivere in un file temporaneo nella directory di destinazione, chiudere/sincronizzare e rinominare solo a trasferimento completato. Prima di partire applicare una policy unica (`ask`, `replace`, `rename`, `skip`) e non unire directory implicitamente.

### 3. Avvio sessione non transazionale

`quickConnect` racchiude chiamata backend, creazione xterm, render e resize nello stesso `try` (`frontend/src/main.js:767-807`). Se il backend ha gia' avviato la shell ma `createSession` o il layout falliscono, l'errore diventa "Could not start local shell" e il backend puo' mantenere una sessione non rappresentata dalla UI. Questo e' compatibile con il messaggio osservato `Cannot access '<simbolo minificato>' before initialization` mentre la shell funziona.

**Correzione:** separare `start backend` da `attach UI`. Se l'attach fallisce, chiamare sempre `StopSSHSession(sessionID)`; mostrare un errore UI distinto e registrare stack/cause non minificate. Aggiungere una guardia affinche' una sessione sia pubblicata nello stato solo dopo inizializzazione completa di xterm.

### 4. Lifecycle dei job file transfer

I job vengono aggiunti a `App.transferJobs` ma mai eliminati (`file_transfer.go:241-365`). Il pulsante Close li rimuove solo dall'array Svelte. Inoltre `ensureNoActiveFileTransferJob` e l'inserimento usano due lock separati, quindi due richieste simultanee possono avviare due job per la stessa risorsa.

**Correzione:** introdurre `DismissFileTransferJob` e una retention backend limitata; rendere check+insert un'unica sezione critica. Alla chiusura sessione attendere la terminazione del job (con timeout) prima di chiudere SFTP.

### 5. Confini filesystem e move assoluto

`localPath` e `remotePath` fanno un controllo lessicale, ma le operazioni successive seguono symlink (`file_transfer.go:1211-1230`). Un link dentro la home/remote root puo' uscire dal perimetro. Inoltre `StartFileTransferUploadJob` accetta path assoluti e, con `Move`, esegue `os.RemoveAll(sourcePath)` (`file_transfer.go:280-332`, `801-835`).

**Correzione:** decidere esplicitamente se il file manager deve essere confinato. Se si', risolvere e verificare i symlink a ogni operazione; per upload esterni usare handle/path autorizzati dalla selezione nativa e vietare `Move` sui path assoluti non appartenenti al workspace locale.

### 6. Fingerprint obsoleta dopo Edit

`UpdateResource` modifica host/IP/porta ma mantiene `HostKeyFingerprint` (`internal/application/service.go:149-186`). Cambiare destinazione porta a un mismatch inevitabile oppure associa la fiducia del vecchio endpoint al nuovo record.

**Correzione:** se cambia uno tra host/IP/porta, azzerare la fingerprint e richiedere nuovamente TOFU. Mostrare questa conseguenza nel form Edit e coprirla con test.

### 7. Invarianti runtime affidate alla UI

`DeleteResource` nel backend ferma i tunnel, ma non sessioni SSH e file transfer (`app.go:170-180`); sono chiusi dal frontend prima della delete. `sshSession` non conserva neppure `resourceID`. Manca inoltre un hook `OnShutdown` in `main_desktop.go` per chiudere esplicitamente shell, tunnel e transfer.

**Correzione:** creare un piccolo `RuntimeManager` proprietario delle risorse per `resourceID`. Delete, import e shutdown devono passare da un'unica operazione backend idempotente; il frontend deve soltanto rifletterne gli eventi.

### 8. Esposizione tunnel

Il bind accetta indirizzi non loopback; SOCKS5 supporta solo modalita' senza autenticazione (`app.go:1203-1225`, `internal/remotessh/socks.go:47-103`). Le fasi di handshake/proxy non impostano deadline, quindi client incompleti possono trattenere goroutine.

**Correzione:** mantenere loopback come default vincolante e chiedere conferma esplicita per bind pubblici, chiarendo che il proxy e' senza autenticazione. Aggiungere timeout di handshake e cancellazione/chiusura delle connessioni attive insieme al tunnel.

### 9. Moduli troppo grandi e fallback demo

`frontend/src/main.js` supera 3.500 righe e contiene markup, stato, API, terminali, modali, auth e demo; anche `styles.css` supera 1.900 righe senza un livello chiaro di variabili/componenti. `app.go` supera 2.200 righe e mescola controller Wails, SSH, import/export, chiavi e update check. Se i binding Wails mancano, il frontend usa dati demo e operazioni finte invece di fallire chiaramente.

**Refactor incrementale:** estrarre prima moduli senza cambiare framework: `api`, `state/session-store`, `terminal`, `resources`, `auth`, `dialogs`, `updates`; raccogliere colori e dimensioni ricorrenti in custom properties CSS. Nel backend separare controller `keys`, `database`, `sessions`, `tunnels`, `updates`; mantenere `App` come facade Wails. Abilitare il demo solo con flag di sviluppo esplicito.

### 10. Copertura insufficiente sulle aree piu' fragili

I test Go coprono bene datastore, dominio e parser, ma quasi nulla del lifecycle concorrente di sessioni/tunnel/job. Il frontend non ha script di test, lint o type-check (`frontend/package.json`).

**Correzione:** aggiungere test Go con fake shell/client per close, keepalive, delete/import e job concorrenti; eseguire `go test -race ./...`. Dopo l'estrazione dei moduli frontend, usare Vitest per selezione/tab/auth/error mapping e pochi test headless per focus, modal e bootstrap locale.

### 11. Contratti duplicati e stringhe sentinella

I quattro form di autenticazione e i relativi mapper sono duplicati tra `main.js` e `FileTransferApp.svelte`. Gli errori host key sono stringhe `BASHES_HOST_KEY_*` analizzate con regex (`frontend/src/main.js:2441-2526`); anche i livelli del log sono inferiti dal testo.

**Refactor:** definire DTO/eventi con `code`, `message` e `details`; condividere un modulo frontend per auth e classificazione errori. Questo riduce branching e impedisce che una modifica al testo rompa il comportamento.

### 12. Hardening CI e installer

Il workflow dichiara `contents: write` globalmente (`.github/workflows/build-desktop.yml:13-14`), quindi anche job e action che devono solo leggere ricevono un token piu' potente. Le release e lo script Linux non verificano checksum degli asset.

**Correzione:** impostare `contents: read` di default e `contents: write` solo sul job release; valutare pin delle action a commit SHA. Generare `SHA256SUMS` nel job release e verificarlo nello script Linux prima di installare il binario.

### 13. Piccole transazioni su disco

`SaveSSHKeySettings` usa un nome `.tmp` fisso e una sequenza propria (`app.go:644-676`), mentre `GenerateSSHKey` puo' lasciare la privata senza pubblica se il secondo write fallisce (`app.go:787-826`). Esistono quindi tre implementazioni simili di scrittura atomica.

**Refactor:** riusare un helper atomico unico; per le coppie di chiavi scrivere entrambi i temporanei con creazione esclusiva, poi pubblicarli e ripulire tutto su errore.

### 14. Duplicazione nel packaging

Il workflow ripete setup e packaging per cinque target e la versione e' replicata in `wails.json`, `package.json`, lockfile e tag. Le release note sono hardcoded nel workflow e richiedono una modifica al codice a ogni rilascio.

**Refactor:** usare matrix/reusable workflow dove le differenze di piattaforma lo consentono; aggiungere uno script unico di bump/verifica versione e generare le note dal tag/changelog o da un file release dedicato.

## Ordine consigliato

1. Correggere 1, 2 e 3: sicurezza SSH, integrita' dei file e bootstrap sessione.
2. Consolidare lifecycle runtime con 4 e 7, includendo test concorrenti.
3. Applicare hardening 5, 6, 8 e 12.
4. Estrarre moduli frontend/backend (9 e 11) in piccoli commit senza redesign UI.
5. Aggiungere la copertura del punto 10 durante ogni estrazione; chiudere con 13 e 14.

## Aspetti solidi da preservare

- Datastore JSON versionato, validato, con backup e scrittura atomica.
- Separazione gia' presente tra `domain`, `application`, `store`, `remotessh` e `localterm`.
- TOFU con fingerprint e supporto `known_hosts`, una volta eliminata la persistenza del bypass.
- Output terminale trasportato come byte/base64 e resize PTY.
- File transfer a blocchi nel backend con progresso, cancellazione e UI Svelte isolata.

## Verifiche eseguite dopo i fix

- `go test ./...`
- `go test -race ./...`
- `go vet ./...`
- `npm test --prefix frontend`
- `npm run build --prefix frontend`
- build Wails `linux/amd64` con tag `desktop,webkit2_41`
- validazione YAML del workflow e sintassi degli script shell
- verifica della coerenza versione con `node scripts/version.mjs check`
