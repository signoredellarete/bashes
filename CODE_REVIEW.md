# Code Review — Bashes

Data: 2026-07-03
Ambito: revisione completa di backend Go, frontend (main.js, Svelte file transfer, CSS) e valutazione UI/UX rispetto all'obiettivo del progetto (alternativa semplice e leggera a MobaXterm).

Il documento è organizzato per priorità. Ogni voce indica: dove sta il problema, perché è un problema, e istruzioni concrete per chi (persona o agente AI) dovrà implementare la correzione.

---

## 1. Riepilogo delle priorità

| # | Problema | Tipo | Priorità |
|---|----------|------|----------|
| 2.1 | La verifica della chiave host del server è di fatto disattivata | Sicurezza | Alta |
| 2.2 | Copia di una cartella dentro sé stessa causa un ciclo infinito | Bug | Alta |
| 2.3 | Caratteri accentati/Unicode possono arrivare corrotti nel terminale | Bug | Alta |
| 2.4 | Scritture concorrenti su hosts.json possono perdere dati | Bug | Alta |
| 2.5 | Percorso chiavi errato (`data/keys`) in un ramo di fallback | Bug | Media |
| 2.6 | Upload di file grandi passa interamente in memoria come base64 | Bug/Design | Media |
| 2.7 | Trasferimenti file bloccanti, senza progresso né annullamento | Design | Media |
| 2.8 | Nessun keepalive SSH: sessioni e tunnel cadono in silenzio | Design | Media |
| 2.9 | Comando remoto di installazione chiave con concatenazione fragile | Bug minore | Bassa |
| 3.1 | Chiusura sessione: l'output finale del terminale viene perso | UX/Bug | Alta |
| 3.2 | Blocco globale di tutti i controlli durante ogni operazione | UX/Bug | Media |
| 3.3 | La copia negli appunti può non funzionare su Linux | Bug | Media |
| 3.4 | Errori mostrati solo in una riga di stato facilissima da perdere | UX | Alta |
| 3.5 | Trucco manuale per riservare colonne al terminale | Bug minore | Bassa |
| 3.6 | Messaggio fuorviante quando la connessione rapida fallisce | UX | Bassa |
| 4.x | Migliorie UI/UX (scorciatoie, pannello connessione, tunnel, empty state, ecc.) | UX | Varie |

---

## 2. Bug e problemi di design nel backend (Go)

### 2.1 La verifica della chiave host è di fatto disattivata — PRIORITÀ ALTA (sicurezza)

**Dove:** `frontend/src/main.js` (funzione `quickConnect`, riga ~545, e i form Connect/Tunnel/Keys con la checkbox `trustHostKey` spuntata di default), `app.go` funzione `hostKeyPolicy` (riga ~950), `internal/remotessh/client.go` funzione `HostKeyCallback`.

**Problema:** il doppio click su un host chiama `quickConnect` che passa sempre `trustHostKey: true`, e in tutti i form la checkbox "Trust host key" è spuntata di default. `trustHostKey: true` si traduce in `ssh.InsecureIgnoreHostKey()`: il client accetta qualunque server senza verificarne l'identità. In pratica l'app non protegge mai dall'attacco "man in the middle", che è esattamente ciò che la verifica della chiave host previene. Inoltre, se l'utente toglie la spunta, il comportamento è pessimo: se l'host non è nel file `known_hosts` la connessione fallisce con un errore tecnico incomprensibile, e se `known_hosts` non esiste l'errore è "ssh host key policy is required".

**Come correggere (istruzioni per l'agente):**
1. Aggiungere al modello dati (`internal/domain/model.go`) un campo per memorizzare l'impronta della chiave host, ad esempio `HostKeyFingerprint string` dentro `Auth` o direttamente su `Host`/`Endpoint`.
2. Implementare il modello "trust on first use" (lo stesso di OpenSSH):
   - Alla prima connessione, se non c'è un'impronta salvata, il backend calcola l'impronta SHA256 della chiave del server (`ssh.FingerprintSHA256`) e la restituisce al frontend con un errore tipizzato (es. errore che contiene `fingerprint` e `hostname`).
   - Il frontend mostra un dialogo: "Prima connessione a X. Impronta del server: SHA256:... Vuoi fidarti?" con pulsanti Accetta/Rifiuta. Se l'utente accetta, la connessione viene ripetuta con un flag "accetta e salva" e il backend salva l'impronta nel datastore.
   - Alle connessioni successive la callback host key confronta la chiave ricevuta con l'impronta salvata: se coincide procede, se non coincide fallisce con un messaggio chiaro ("La chiave del server è cambiata: possibile attacco o server reinstallato").
3. Rimuovere la checkbox "Trust host key" dai form (o lasciarla come opzione avanzata esplicitamente etichettata come insicura) e rimuovere `trustHostKey: true` da `quickConnect`.
4. Mantenere in aggiunta il supporto a `~/.ssh/known_hosts` come verifica alternativa quando l'impronta salvata non c'è.

### 2.2 Copiare una cartella dentro sé stessa causa un ciclo infinito — PRIORITÀ ALTA

**Dove:** `file_transfer.go`, funzioni `copy` (riga ~420) e `copyUnlocked` (riga ~448).

**Problema:** `copyUnlocked` prima crea la cartella di destinazione, poi elenca i figli della sorgente e li copia ricorsivamente. Se la destinazione sta dentro la sorgente (es. trascino la cartella `/local/a` sopra sé stessa o su una sua sottocartella: destinazione `/local/a/a`), la cartella appena creata compare nell'elenco dei figli della sorgente e viene copiata a sua volta, creando `a/a/a`, `a/a/a/a`, ... all'infinito, fino a riempire il disco o mandare in errore la sessione. Con il file manager a doppio pannello e il drag & drop questo scenario è facile da innescare per sbaglio.

**Come correggere:** in `copy()` (prima di chiamare `copyUnlocked`) aggiungere un controllo: se `sourceScope == targetScope` e il percorso di destinazione (`destinationRel`) è uguale alla sorgente (`sourceRel`) oppure inizia con `sourceRel + "/"`, restituire un errore chiaro tipo "cannot copy a folder into itself". Aggiungere un test in `file_transfer_test.go` che verifichi questo caso.

### 2.3 Output del terminale: caratteri multibyte possono arrivare corrotti — PRIORITÀ ALTA

**Dove:** `app.go`, tipo `eventWriter` (riga ~716).

**Problema:** l'output SSH arriva a blocchi di byte arbitrari. `eventWriter.Write` converte ogni blocco in stringa e lo emette come evento. Un carattere UTF-8 multibyte (lettere accentate, caratteri di disegno riquadri usati da `htop`, `mc`, `top` con locale UTF-8) può essere spezzato tra due blocchi: la conversione produce caratteri di sostituzione (�) e il terminale mostra testo corrotto. Il problema è intermittente e dipende dal punto in cui il flusso viene spezzato, quindi difficilissimo da diagnosticare per un utente.

**Come correggere:** rendere `eventWriter` un tipo con stato (puntatore) che mantiene un piccolo buffer dei byte finali incompleti: a ogni `Write`, accodare i byte al buffer, individuare il prefisso valido UTF-8 più lungo (con `utf8.DecodeLastRune`/`utf8.Valid` sugli ultimi 1–3 byte), emettere solo quello e conservare il resto per la chiamata successiva. In alternativa (più semplice e robusta): trasmettere i dati come base64 nell'evento e decodificarli nel frontend passando i byte grezzi a xterm.js (`terminal.write(Uint8Array)`), che gestisce da solo l'UTF-8 spezzato. La seconda strada è preferibile perché elimina il problema alla radice.

### 2.4 Accesso concorrente al datastore senza lock — PRIORITÀ ALTA

**Dove:** `internal/application/service.go` (tutte le operazioni) e `internal/store/repository.go`.

**Problema:** ogni operazione fa "leggi hosts.json → modifica in memoria → riscrivi tutto". Wails esegue le chiamate dal frontend su goroutine separate, e il backend stesso chiama `SetResourceAuth` durante l'avvio di sessioni/tunnel. Due operazioni sovrapposte (es. l'utente rinomina un host mentre una connessione appena avviata salva la preferenza di autenticazione) possono leggere lo stesso stato e l'ultima scrittura cancella la modifica dell'altra. Il file resta valido, ma si perdono dati in modo silenzioso.

**Come correggere:** aggiungere un `sync.Mutex` al tipo `Service` e prenderlo all'inizio di ogni metodo che fa load+save (`AddHost`, `AddSubsystem`, `UpdateResource`, `DeleteResource`, `ReorderHosts`, `SetResourceAuth`). È una modifica piccola e sufficiente, dato che il datastore è unico e le operazioni sono rapide.

### 2.5 Percorso chiavi sbagliato in un ramo di fallback — PRIORITÀ MEDIA

**Dove:** `app.go`, funzione `authMethods` (riga ~860): `keyPath = filepath.Join("data", "keys", sanitizeKeyName(input.KeyName))`.

**Problema:** se `authMethods` viene chiamata con un `KeyName` ma senza `PrivateKeyPath` risolto, costruisce il percorso `data/keys/<nome>` relativo alla directory di lavoro corrente, che non è la directory dati reale dell'app (`~/.local/share/bashes/keys`, ecc.). Oggi i chiamanti principali passano da `resolveSessionKeyPath` che risolve il percorso corretto prima, quindi il ramo è quasi sempre inattivo — ma è una trappola: qualunque nuovo chiamante che passi solo `KeyName` fallirà con "read private key: no such file or directory".

**Come correggere:** eliminare questo fallback da `authMethods` e far sì che la funzione restituisca un errore esplicito se riceve `KeyName` senza `PrivateKeyPath` (oppure spostare la risoluzione dentro `dialResource`, che ha accesso all'`App` e quindi a `keysDir()`). L'obiettivo è avere un solo punto che traduce nome chiave → percorso.

### 2.6 Upload da drag & drop: file interi in memoria come base64 — PRIORITÀ MEDIA

**Dove:** `frontend/src/file-transfer/api.js` (`fileToBase64`), `file_transfer.go` (`FileTransferCreateInput.Data`, `writeRemoteFile`, `writeLocalFile`).

**Problema:** quando si trascina un file dal sistema operativo nel file manager, il file viene letto per intero in memoria nel browser, convertito in base64 concatenando stringhe (molto costoso), passato attraverso il bridge JSON di Wails e decodificato in Go sempre in memoria. Con file da centinaia di MB l'app rallenta pesantemente o va in crash; non c'è indicazione di progresso.

**Come correggere:** implementare un caricamento a blocchi: aggiungere al backend tre metodi (`BeginFileUpload` che apre il file di destinazione e restituisce un ID, `WriteFileUploadChunk(id, base64chunk)` che accoda un blocco, `FinishFileUpload(id)` che chiude, più un `AbortFileUpload`). Nel frontend, leggere il `File` con `file.slice(offset, offset+chunkSize)` (es. blocchi da 1–4 MB), inviare i blocchi in sequenza e aggiornare una barra di progresso. Questo risolve memoria e progresso insieme.

### 2.7 Trasferimenti bloccanti, senza progresso né annullamento — PRIORITÀ MEDIA

**Dove:** `file_transfer.go` (`copy`, `copyUnlocked`), `frontend/src/file-transfer/FileTransferApp.svelte`.

**Problema:** copia e spostamento avvengono nella chiamata Wails sincrona, con il mutex della sessione di trasferimento tenuto per tutta la durata. Per una cartella grande: la finestra mostra solo "Working...", non si può annullare, e qualsiasi altra operazione sul file manager resta in coda. Se l'utente chiude la modale a metà, la connessione viene chiusa sotto i piedi della copia e il risultato è un trasferimento parziale senza spiegazione.

**Come correggere:**
1. Eseguire le copie in una goroutine: il metodo `CopyFileTransferItems` restituisce subito un ID operazione; il progresso viene emesso con eventi Wails (`transfer:progress` con byte copiati/totali e file corrente, `transfer:done`, `transfer:error`).
2. Passare un `context.Context` annullabile lungo la catena di copia (usare `io.Copy` su un reader che controlla `ctx.Err()`), e aggiungere un metodo `CancelFileTransferOperation(id)`.
3. Nel frontend mostrare una barra di progresso con pulsante Annulla, e chiedere conferma se l'utente chiude la modale con un'operazione in corso.

### 2.8 Nessun keepalive SSH — PRIORITÀ MEDIA

**Dove:** `app.go` (`StartSSHSession`, `StartSSHTunnel`), `internal/remotessh`.

**Problema:** le connessioni SSH non inviano keepalive. Dietro NAT/firewall una sessione o un tunnel inattivi vengono chiusi dopo alcuni minuti: il terminale sembra congelato finché non si preme un tasto, e i tunnel muoiono senza che la spia "tun" nella sidebar cambi in tempo utile. Per un'app che vuole sostituire MobaXterm (che invia keepalive di default) è una mancanza sentita.

**Come correggere:** dopo la creazione del client SSH, avviare una goroutine che ogni 30 secondi invia `client.SendRequest("keepalive@openssh.com", true, nil)`; se la richiesta fallisce due volte di seguito, chiudere il client (le logiche esistenti di `waitForShell`/`serveTunnel` faranno il resto, notificando il frontend). Fermare la goroutine con il context già presente nelle strutture `sshSession`/`sshTunnel`.

### 2.9 Concatenazione fragile nel comando di installazione della chiave — PRIORITÀ BASSA

**Dove:** `app.go`, `InstallSSHKey` (riga ~253).

**Problema:** il comando remoto è `mkdir ... && chmod ... && touch ... && grep -qxF ... || printf ... >> ... && chmod ...`. In shell `&&` e `||` hanno la stessa precedenza e si valutano da sinistra: se uno qualunque dei comandi iniziali fallisce (es. `mkdir` per permessi), il `printf` viene eseguito comunque, scrivendo su un file in una situazione anomala; inoltre l'esito complessivo può risultare "successo" mascherando l'errore vero.

**Come correggere:** raggruppare esplicitamente la parte condizionale: `mkdir -p ~/.ssh && chmod 700 ~/.ssh && touch ~/.ssh/authorized_keys && chmod 600 ~/.ssh/authorized_keys && { grep -qxF <chiave> ~/.ssh/authorized_keys || printf '%s\n' <chiave> >> ~/.ssh/authorized_keys; }`. Così il `||` si applica solo al `grep`.

### 2.10 Osservazioni minori backend (da sistemare quando capita)

- `app.go`, `dataDirForOS`: l'ultimo fallback è `filepath.Join("data")`, una directory relativa alla cwd. Meglio fallire con un errore chiaro all'avvio piuttosto che scrivere dati in una posizione imprevedibile.
- `file_transfer.go`: i symlink non sono gestiti; `Stat` li segue, quindi un link a una cartella superiore può creare ricorsioni in copia/cancellazione remota. Usare `Lstat` e copiare i link come link (o saltarli con un avviso).
- `app.go`, `DeleteResource`: il backend ferma i tunnel della risorsa cancellata ma non le sessioni SSH; oggi ci pensa il frontend, ma per robustezza conviene fermare anche le sessioni nel backend (specularmente a `stopTunnelsForResource`).
- `GenerateSSHKey` non supporta una passphrase per la chiave generata: aggiungere un campo opzionale e usare `ssh.MarshalPrivateKeyWithPassphrase`.
- `defaultPrivateKeyAuthMethods` prova la stessa passphrase su tutte le chiavi di default: innocuo, ma se una chiave di default è cifrata e la passphrase non corrisponde, la chiave viene ignorata in silenzio. Va bene, purché il messaggio d'errore finale suggerisca le cause possibili.

---

## 3. Bug e fragilità nel frontend

### 3.1 Alla chiusura della sessione SSH l'output del terminale viene distrutto — PRIORITÀ ALTA

**Dove:** `frontend/src/main.js`, gestore `ssh:closed` in `registerSSHEvents` (riga ~1786) che chiama `removeSessionFromUI`.

**Problema:** quando la sessione termina (anche per un errore: rete caduta, `exit` del server, autenticazione scaduta), la scheda e il terminale vengono rimossi immediatamente. L'utente perde le ultime righe di output, che spesso contengono proprio la spiegazione della disconnessione. Esiste già uno stato `closed` con relativo stile CSS (`.session-tab.closed`), ma non viene mai usato: il codice rimuove tutto e basta.

**Come correggere:** nel gestore `ssh:closed`, invece di chiamare `removeSessionFromUI`, marcare la sessione come chiusa (`session.closed = true`), scrivere nel terminale una riga finale ben visibile (es. `\r\n[Sessione chiusa: <motivo>]`), disabilitare l'input (`terminal.options.disableStdin = true`) e rieseguire `renderTabs()`. La scheda resta consultabile con l'etichetta "closed" e si chiude solo con la X (che già chiama `stopSession` → `removeSessionFromUI`). Aggiungere sulla scheda chiusa un pulsante o doppio click "Riconnetti" che riusa `quickConnect` sulla stessa risorsa.

### 3.2 Blocco globale di tutti i controlli durante ogni operazione — PRIORITÀ MEDIA

**Dove:** `frontend/src/main.js`, `withBusy` e `setDisabledState` (riga ~1678).

**Problema:** ogni operazione disabilita **tutti** i bottoni, input, select e textarea della pagina e poi li riabilita tutti. Effetti collaterali: (1) durante una connessione lenta l'intera interfaccia sembra rotta; (2) alla fine, `disabled = false` riabilita anche controlli che dovrebbero restare disabilitati finché `renderSelection()` non li ricalcola, con possibili lampeggi o stati incoerenti nei pannelli; (3) `if (state.busy) return` scarta in silenzio i click dell'utente, che non capisce perché il pulsante "non funziona".

**Come correggere:** sostituire il blocco globale con: (a) un flag `busy` che disabilita solo il pulsante che ha avviato l'operazione, mostrandoci sopra uno spinner o testo "Connecting…"; (b) per le operazioni con pannello aperto, disabilitare solo i controlli di quel pannello; (c) quando un'azione viene rifiutata perché un'altra è in corso, mostrare un avviso ("Operazione in corso, attendi"). Eliminare `setDisabledState` e affidare lo stato dei pulsanti dell'header al solo `renderSelection()`.

### 3.3 La copia negli appunti può non funzionare su Linux — PRIORITÀ MEDIA

**Dove:** `frontend/src/main.js`, `writeClipboard`/`readClipboard` (riga ~1762).

**Problema:** l'app usa `navigator.clipboard`, che su WebKitGTK (il motore usato da Wails su Linux) è spesso assente o negato. I fallimenti sono inghiottiti da `.catch(() => {})`: la selezione sembra copiare ma gli appunti restano vuoti, e il click destro non incolla nulla, senza alcun messaggio.

**Come correggere:** usare le API clipboard di Wails come prima scelta: `window.runtime.ClipboardSetText(text)` e `ClipboardGetText()` (già disponibili nel runtime v2), con `navigator.clipboard` come fallback. Se entrambe falliscono, mostrare un avviso una tantum con `writeNotice`. Verificare il funzionamento su Linux e Windows.

### 3.4 Gli errori compaiono solo in una riga di stato nel footer — PRIORITÀ ALTA (UX)

**Dove:** `frontend/src/main.js`, `writeNotice`; markup `#app-status` nel footer.

**Problema:** ogni messaggio — dal banale "Host order updated" al critico "Error: ssh: unable to authenticate" — finisce nella stessa riga piccola in basso a sinistra, dove il messaggio successivo sovrascrive il precedente. Un errore di connessione può sparire in un attimo, e l'utente non ha modo di rileggerlo.

**Come correggere:** introdurre un piccolo sistema di notifiche a comparsa (toast) senza librerie esterne: un contenitore fisso in alto a destra; `notify(message, level)` crea un elemento con stile diverso per `info`/`success`/`error`; gli errori restano visibili finché non vengono chiusi (o almeno 10 secondi), le info spariscono dopo 3–4 secondi. Mantenere la riga di stato per i messaggi passivi (es. stato tunnel), ma instradare tutti gli errori sui toast. In più, tenere un log degli ultimi N messaggi consultabile da un'icona nel footer.

### 3.5 Ridimensionamento del terminale con correzioni manuali — PRIORITÀ BASSA

**Dove:** `frontend/src/main.js`, `fitActiveTerminal` e `terminalScrollbarReserveColumns` (riga ~1717); CSS `.terminal-pane { padding: 10px 0 0 10px; }`.

**Problema:** dopo il calcolo di `fitAddon.fit()`, il codice toglie a mano 3+ colonne per "riservare" spazio alla scrollbar e una riga in basso. È un cerotto su un problema di layout: il pannello non ha padding a destra/in basso, quindi il testo toccherebbe la scrollbar. Il risultato è un terminale sempre più stretto del necessario e un valore di colonne che non corrisponde alla larghezza reale, con possibili impaginazioni strane in applicazioni a tutto schermo (vim, htop).

**Come correggere:** dare al pannello un padding simmetrico (es. `padding: 10px 14px 10px 10px`) e lasciare che `fitAddon.fit()` calcoli da solo colonne e righe, eliminando `terminalScrollbarReserveColumns` e le sottrazioni manuali in `fitActiveTerminal`. Verificare con `htop` e `vim` che l'ultima colonna e l'ultima riga siano visibili.

### 3.6 Messaggio fuorviante quando la connessione rapida fallisce — PRIORITÀ BASSA

**Dove:** `frontend/src/main.js`, `quickConnect` (riga ~552).

**Problema:** qualsiasi errore (host irraggiungibile, DNS errato, timeout, chiave host) produce "Connection needs credentials: …" e apre il pannello credenziali, suggerendo all'utente che il problema sia la password anche quando non lo è.

**Come correggere:** replicare la distinzione già usata nel file transfer (`isAuthError` in `FileTransferApp.svelte`): se il messaggio contiene "authenticate"/"permission denied"/"no supported methods" aprire il pannello credenziali; altrimenti mostrare l'errore com'è ("Connessione fallita: …") senza aprire il pannello. Estrarre `isAuthError` in un modulo condiviso.

### 3.7 Osservazioni minori frontend

- La dimensione del font del terminale non viene salvata (la sidebar compressa sì): salvarla in `localStorage` accanto a `bashes.sidebarCollapsed` e ripristinarla all'avvio.
- Il tasto Escape chiude solo il modale di conferma; deve chiudere anche i pannelli laterali (Connect, Tunnel, Keys, resource) e la modale file transfer. Aggiungere un gestore unico che chiude il livello più in alto aperto.
- I pannelli e le modali non intrappolano il focus (con Tab si finisce sotto lo scrim) e i pannelli laterali non hanno `role="dialog"`/`aria-modal`. Sistemare per l'accessibilità da tastiera.
- Nel filtro di ricerca della sidebar, se corrisponde solo un sottosistema, il padre scompare e la riga figlia appare da sola fuori contesto. Mantenere visibili gli antenati delle righe corrispondenti.
- `main.js` è un file unico da ~2000 righe con dentro anche l'implementazione demo. Suddividerlo in moduli (`state.js`, `api.js`, `sidebar.js`, `terminal.js`, `panels.js`, `demo.js`) senza cambiare comportamento: renderà molto più sicuri gli interventi futuri, inclusi quelli di questo report.
- In `main_desktop.go` la finestra non ha dimensioni minime: con finestre molto piccole il layout si rompe. Aggiungere `MinWidth: 940, MinHeight: 600` alle opzioni Wails.

---

## 4. Revisione UI/UX (rispetto all'obiettivo "MobaXterm semplice e leggero")

L'interfaccia è pulita e coerente nei colori, e il flusso base (aggiungi host → doppio click → terminale) funziona bene. I punti sotto sono ordinati per impatto percepito da un utente che arriva da MobaXterm/Termius.

### 4.1 Scorciatoie da tastiera — impatto alto

Oggi non esiste alcuna scorciatoia. Per un'app di terminali è la mancanza più sentita: l'utente vive sulla tastiera.

**Implementazione suggerita:** un gestore `keydown` globale (che ignori gli eventi quando il focus è dentro il terminale, tranne le combinazioni con Ctrl+Shift) con:
- `Ctrl+Shift+T` / doppio click sull'host selezionato: nuova sessione sulla risorsa selezionata.
- `Ctrl+PgUp` / `Ctrl+PgDown` (o `Ctrl+Shift+Tab`/`Ctrl+Tab`): scheda precedente/successiva.
- `Ctrl+Shift+W`: chiudi scheda attiva.
- `Ctrl+Shift+C` / `Ctrl+Shift+V`: copia/incolla nel terminale (xterm.js non intercetta di default `Ctrl+Shift+C`; usare `terminal.attachCustomKeyEventHandler`).
- `Ctrl+Shift+F`: ricerca nel buffer del terminale, aggiungendo la dipendenza `@xterm/addon-search` e una piccola barra di ricerca sopra il terminale.
- `Ctrl+K` o `/`: focus sulla ricerca host della sidebar.
Documentare le scorciatoie in un tooltip o in un pannello "?".

### 4.2 Pannello di connessione: scegliere il metodo, non compilare quattro campi — impatto alto

Il form Connect mostra insieme "Bashes Key", "Password", "Private Key Path" e "Key Passphrase". Non è chiaro quale campo serva né cosa succede se se ne compilano due (il backend ha una precedenza interna che l'utente non conosce).

**Implementazione suggerita:** in cima al form un selettore "Metodo di autenticazione" (radio o select) con voci: `Password`, `Chiave Bashes`, `File di chiave`, `Agent/chiavi di sistema`. Mostrare solo i campi del metodo scelto. Preselezionare il metodo dalla preferenza salvata (`resource.auth.method`, già disponibile) — oggi la preferenza "password" salvata non preseleziona nulla. Stesso trattamento per il form Tunnel (che duplica gli stessi campi: estrarre un blocco riutilizzabile).

### 4.3 Menu contestuale sulla sidebar — impatto alto

Tutte le azioni (Edit, Add Subsystem, Keys, Delete, Files, Tunnel, Connect) vivono nell'header in alto e agiscono "sulla risorsa selezionata": l'associazione è poco evidente e richiede due passaggi (seleziona, poi cerca il bottone in alto a destra).

**Implementazione suggerita:** click destro su una riga della sidebar apre un menu contestuale con: Connect / New session, Edit, Add subsystem, Files, Tunnel, Install key, Delete (in rosso, separato). Il menu è un semplice `div` posizionato alle coordinate del mouse, chiuso su click fuori o Escape. Le azioni riusano le funzioni esistenti (`openEditPanel`, `deleteSelectedResource`, ecc.) dopo aver impostato `state.selectedId`. Mantenere anche i bottoni nell'header per la scopribilità.

### 4.4 Stato vuoto e primo avvio — impatto medio

Al primo avvio si vede un'enorme area nera con "No active terminal session" e una sidebar vuota. Nessuna guida.

**Implementazione suggerita:** quando non ci sono host, mostrare nell'area principale un pannello di benvenuto: logo, una riga di spiegazione, un pulsante primario "Add your first host" (apre il pannello esistente) e due righe su come connettersi (doppio click) e generare una chiave. Quando ci sono host ma nessuna sessione, il testo dovrebbe suggerire l'azione: "Double-click a host in the sidebar to connect".

### 4.5 Gestione tunnel: vista unica invece di uno-per-risorsa — impatto medio

Il backend supporta più tunnel per risorsa, ma la UI ne permette uno solo (`tunnelForResource` prende il primo) e per vedere/fermare un tunnel bisogna selezionare la risorsa giusta e aprire il pannello. La spia "tun" nella sidebar non dice quale porta è attiva.

**Implementazione suggerita:** trasformare il pannello Tunnel in due sezioni: sopra, il form di creazione (già esistente); sotto, la lista di **tutti** i tunnel attivi (`apiListSSHTunnels`) con tipo, `localAddress → forwardTarget`, risorsa e pulsante Stop per riga. Rimuovere il blocco "Tunnel already active" in `submitTunnel` per consentire più tunnel sulla stessa risorsa. Aggiornare la lista quando arriva l'evento `ssh:status` con "Tunnel closed". Nel chip "tun" della sidebar, mostrare nel tooltip l'elenco delle porte attive.

### 4.6 File transfer: riusare la sessione e integrare meglio — impatto medio

`StartFileTransfer` apre una nuova connessione SSH anche quando esiste già una sessione attiva verso la stessa risorsa: l'utente con autenticazione a password deve reinserirla per aprire i file. Inoltre la modale copre tutto il terminale.

**Implementazione suggerita:** (1) nel backend, se esiste già una `sshSession` per la stessa risorsa, riusare il suo `*ssh.Client` per creare il client SFTP invece di comporre una nuova connessione (attenzione a non chiudere il client condiviso in `CloseFileTransfer`: aggiungere un flag "client posseduto/condiviso" nella `fileTransferSession`). (2) valutare in seguito una vista SFTP agganciata di lato al terminale (stile MobaXterm) invece della modale; nel breve periodo basta il riuso della connessione.

### 4.7 Aspetto visivo — impatto medio

- **Tema misto:** sidebar e header chiari, area terminale scura. Funziona, ma le schede scure sopra lo sfondo chiaro dell'header staccano bruscamente. Proposta minima: offrire un tema scuro completo (variabili CSS per i colori già usati: `--bg`, `--panel`, `--border`, `--accent`, ecc., con `data-theme="dark"` sul root e un interruttore nelle impostazioni). La palette scura esiste già di fatto nei colori del terminale.
- **Testo al posto delle icone:** i pulsanti "X", "<", ">", "+", "A -/+" sono lettere. Sostituirli con icone SVG inline (una manciata: chiudi, comprimi, aggiungi, font +/-) migliora molto la percezione di cura. Nessuna libreria: 5–6 SVG di 16px incollati come stringhe.
- **Tipo di risorsa:** il riquadro "HOST/VM/LXC/DOCKER" a sinistra di ogni riga è funzionale; si può rafforzare con un colore per tipo (host teal, vm blu, lxc arancio, docker azzurro) così l'occhio distingue i tipi senza leggere.
- **Riga di stato:** oggi il footer è quasi vuoto; con i toast del punto 3.4 può ospitare informazioni utili permanenti: numero sessioni attive, numero tunnel attivi, versione dell'app.

### 4.8 Impostazioni minime dell'app — impatto medio

Non esiste un pannello impostazioni. Le richieste tipiche che arriveranno subito: dimensione/famiglia font del terminale, scrollback (oggi fisso al default di xterm, 1000 righe: poco per uso reale — portarlo ad almeno 10.000 configurabile), tema, comportamento della copia su selezione (alcuni la detestano), keepalive on/off.

**Implementazione suggerita:** pannello laterale "Settings" (riusando lo stile `slide-panel`), salvataggio in `localStorage` (o in un `settings.json` accanto a `hosts.json` se si vuole portabilità). Applicare `scrollback` e font alla creazione dei terminali e ai terminali esistenti dove possibile.

### 4.9 Import da ~/.ssh/config — impatto medio (adozione)

Chi prova un'alternativa a MobaXterm ha già decine di host nel proprio `~/.ssh/config`. Farli reinserire a mano è la prima barriera all'adozione.

**Implementazione suggerita:** metodo backend `ImportSSHConfig()` che legge `~/.ssh/config`, estrae `Host` con `HostName`, `Port`, `User`, `IdentityFile` (parser semplice riga per riga, ignorando wildcard e opzioni non gestite) e restituisce un'anteprima; il frontend mostra la lista con checkbox e un pulsante "Importa selezionati" che chiama `AddHost` per ciascuno, impostando `auth` a `path` quando c'è `IdentityFile`.

### 4.10 Osservazioni UX minori

- Incollare con il click destro senza conferma è pericoloso con contenuti multilinea (esecuzione immediata di comandi). Mostrare una conferma quando il testo incollato contiene un a-capo ("Stai incollando N righe").
- Le schede non si riordinano e non si chiudono con il click centrale del mouse: entrambe sono aspettative comuni; il click centrale è a costo quasi zero (`auxclick` sul tab → `stopSession`).
- Il pulsante di eliminazione nel modale di conferma è corretto (rosso, con conteggio dei sottosistemi): bene così. Manca però la stessa cura nel file manager: la cancellazione remota è ricorsiva e irreversibile — verificare che il componente SVAR chieda conferma e, se non lo fa, intercettare `delete-files` con un dialogo proprio.
- Selezionare una risorsa crea sempre una "scheda pendente": è una scelta di design lecita (anteprima della destinazione), ma il click esplorativo nella sidebar muove continuamente l'area schede. Alternativa più calma: mostrare la destinazione solo nell'header e creare la scheda pendente soltanto al doppio click/Connect.
- Tutte le stringhe sono in inglese e sparse nel codice: se si prevede la localizzazione (es. italiano), conviene estrarle presto in un unico oggetto/modulo.

---

## 5. Cosa funziona bene (da non toccare)

- Architettura backend a livelli (`domain` / `application` / `store` / `remotessh`) chiara e testata; datastore JSON con validazione, scrittura atomica e backup: scelta solida e in linea con l'obiettivo "leggero e ispezionabile".
- Migrazione automatica del formato legacy e CLI `bashes-data` per validare/migrare: ottimo.
- Il protocollo SOCKS5 e i forward locali/remoti sono implementati in modo pulito e con test.
- La gestione delle schede (sessioni multiple sulla stessa risorsa, focus history, schede pendenti) è pensata con attenzione.
- La sanificazione degli input (nomi chiave, host, path traversal nel file transfer) è presente e ragionevole.

---

## 6. Ordine di lavoro consigliato

1. **Sicurezza e dati:** 2.1 (host key), 2.4 (lock datastore), 2.2 (copia ricorsiva).
2. **Affidabilità percepita:** 2.3 (UTF-8), 3.1 (schede chiuse), 3.4 (toast errori), 3.3 (clipboard Linux), 2.8 (keepalive).
3. **UX quotidiana:** 4.1 (scorciatoie), 4.2 (pannello connessione), 4.3 (menu contestuale), 3.2 (busy state), 4.5 (tunnel).
4. **File transfer:** 2.6/2.7 (upload a blocchi + progresso), 4.6 (riuso sessione).
5. **Rifiniture:** 4.4, 4.7, 4.8, 4.9 e le osservazioni minori.
