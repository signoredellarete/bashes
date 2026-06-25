# Audit per rinnovo del progetto Bashes

Data audit: 2026-06-25

## Sintesi

Bashes e' una piccola web app locale in PHP, avviata tramite script shell, che espone una UI su `127.0.0.1` e usa il backend per lanciare comandi desktop come `ssh`, `ssh-copy-id`, terminale grafico e file manager. La natura del progetto e' coerente con l'obiettivo originale: uno strumento locale per amministratori Linux, non una web app pubblica.

Il problema principale non e' solo l'eta' di PHP, ma il fatto che input provenienti dal browser e dal file JSON vengano inseriti direttamente in HTML, JavaScript, JSON e comandi shell. Prima di un rinnovo estetico o funzionale serve mettere in sicurezza questi confini.

## Priorita' alta

### 1. Command injection nell'API SSH

File: `public/api/ssh_api.php`

- L'endpoint accetta `func`, `user`, `ip`, `port` dal JSON POST alle righe 11-24.
- Gli stessi valori vengono concatenati in comandi shell alle righe 28, 33 e 38.
- Il comando viene eseguito con `exec()` alla riga 45.

Con l'app in ascolto su localhost il rischio resta concreto: una pagina web aperta nello stesso browser o un processo locale puo' inviare richieste a `127.0.0.1:<porta>/api/ssh_api.php`. Senza validazione, allowlist ed escaping, un valore malevolo in `user`, `ip` o `port` puo' alterare il comando eseguito.

Intervento consigliato:

- Non accettare piu' `user`, `ip` e `port` dal client per azioni operative. Il client dovrebbe inviare solo un ID host/subsystem e il backend dovrebbe rileggere i dati dal datastore locale.
- Validare `func` con una allowlist stretta: `connect`, `remotefs`, `ssh_copy_id`.
- Validare `port` come intero 1-65535.
- Validare host/IP/username con regole compatibili SSH.
- Usare `proc_open()` o `Symfony Process`, passando argomenti separati quando possibile.
- Se resta `exec()`, usare almeno `escapeshellarg()`/`escapeshellcmd()` in modo sistematico.

### 2. XSS persistente e rottura del JavaScript tramite dati salvati

File: `public/hosts.php`, `public/lxc.php`, `public/vm.php`, `public/docker.php`, `public/js/bashes.js`

I dati salvati in `hosts.json` vengono stampati senza escaping:

- attributi HTML: `hostname="<?php echo $host->hostname ?>"` in `public/hosts.php:5`;
- attributi custom: `ref_server="<?php echo $host->hostname ?>"` in `public/hosts.php:20`;
- testo HTML: `<?php echo $host->hostname ?>`, `<?php echo $host->ip ?>`, `<?php echo $host->port ?>` in `public/hosts.php:31-35`;
- stringhe JavaScript inline negli handler `onclick` in `public/hosts.php:46-77`;
- pattern analoghi sono presenti in `lxc.php`, `vm.php` e `docker.php`;
- il JavaScript usa `innerHTML` per dati provenienti dagli attributi DOM in `public/js/bashes.js:13` e `public/js/bashes.js:31`.

Questo consente a un hostname o user salvato di iniettare markup/script o rompere gli handler inline.

Intervento consigliato:

- Introdurre una funzione helper `e($value)` basata su `htmlspecialchars($value, ENT_QUOTES | ENT_SUBSTITUTE, 'UTF-8')`.
- Eliminare gli handler JavaScript inline e passare dati tramite `data-*` oppure ID interni.
- Usare `textContent` invece di `innerHTML` per testi dinamici.
- Centralizzare il rendering delle righe host/subsystem per evitare di correggere quattro copie quasi uguali.

### 3. Scrittura del JSON fragile e potenzialmente corrotta

File: `public/lib/add_host.php`, `public/lib/add_subsystem.php`, `public/lib/delete_host.php`

Problemi osservati:

- `add_host.php` costruisce JSON come stringa alle righe 23-31 e poi lo passa a `json_decode()`. Virgolette o caratteri speciali negli input possono generare `null` o corrompere i dati.
- `add_subsystem.php` ripete lo stesso schema alle righe 23-30.
- `file_put_contents("../db/hosts.json", $json)` viene usato senza `LOCK_EX` in `add_host.php:39`, `add_subsystem.php:40`, `delete_host.php:31`.
- Non c'e' gestione di errori JSON (`json_last_error_msg()`), backup o scrittura atomica.
- `install.sh` crea `public/db/hosts.json` vuoto con `touch` alla riga 118; `index.php` poi fa `foreach ($hosts as $host)` senza garantire che `$hosts` sia un array (`public/index.php:2-3` e `public/index.php:70`).

Intervento consigliato:

- Creare sempre array PHP e poi serializzare con `json_encode()`, senza costruire JSON manualmente.
- Usare funzioni `loadHosts()` e `saveHosts()` condivise.
- Inizializzare il file con `[]`, non vuoto.
- Usare scrittura atomica: scrittura su file temporaneo, `flock()`, `rename()`.
- Gestire JSON non valido mostrando un errore recuperabile e preservando un backup.

### 4. Endpoint mutanti senza protezione locale

File: `public/lib/add_host.php`, `public/lib/add_subsystem.php`, `public/lib/delete_host.php`, `public/api/ssh_api.php`

L'app ascolta solo su `127.0.0.1`, ma non ha alcun token locale, sessione, CSRF token o controllo sull'origine della richiesta. Una pagina web esterna non puo' leggere liberamente le risposte per via delle policy browser, ma puo' spesso provocare richieste cross-site verso servizi su localhost, soprattutto per form POST tradizionali. Per un'app che lancia comandi locali questo confine va rafforzato.

Intervento consigliato:

- Generare un token random all'avvio e inserirlo nelle form e nelle chiamate `fetch`.
- Rifiutare richieste senza token.
- Accettare solo `POST` per azioni mutanti.
- Verificare `Host`/`Origin` quando presenti.
- Restituire status HTTP corretti e JSON strutturato per API.

## Priorita' media

### 5. Dipendenze desktop hardcoded e rilevazione ignorata

File: `install.sh`, `start_bashes.sh`, `public/api/ssh_api.php`, `public/lib/ssh_copy_id.php`

`install.sh` rileva browser, explorer e terminale (`install.sh:64-94`) e li salva in `.env`, ma `public/api/ssh_api.php` usa direttamente `gnome-terminal`, `/usr/bin/nemo`, `/home/sid` alle righe 28, 33 e 38. `public/lib/ssh_copy_id.php` usa `gnome-terminal` e `/home/fabrizio` alla riga 6.

Intervento consigliato:

- Usare davvero le variabili rilevate (`terminal`, `explorer`) oppure una configurazione esplicita.
- Rimuovere path utente hardcoded e usare `$HOME`/directory corrente.
- Supportare almeno `gnome-terminal`, `konsole`, `xfce4-terminal`, `xterm`, `kgx` e `xdg-open` dove possibile.
- Considerare `gio open ssh://...` o `xdg-open ssh://...` per ridurre l'accoppiamento con Nemo.

### 6. Gestione processi fragile negli script di avvio/stop

File: `start_bashes.sh`, `stop_bashes.sh`

Problemi osservati:

- Molte variabili shell non sono quotate (`start_bashes.sh:6-20`, `stop_bashes.sh:6-18`), con rischi su path contenenti spazi o valori mancanti.
- `start_bashes.sh` usa `/usr/bin/php` fisso alla riga 15, mentre il README parla genericamente di `php-cli`.
- Il monitor controlla processi via `ps | grep <pid>` alle righe 33-34: puo' produrre falsi positivi.
- Viene usato `kill -9` alle righe 37 e 44, senza tentativo di terminazione pulita.
- Dopo la rimozione dei pidfile il loop continua e puo' generare errori ripetuti.
- `stop_bashes.sh` assume che `.pid` e `.chrome_pid` esistano e contengano PID validi.

Intervento consigliato:

- Aggiungere `set -euo pipefail` dove compatibile.
- Quotare sempre le variabili.
- Usare `command -v php`.
- Usare `kill -0 "$pid"` per verificare processi.
- Provare prima `TERM`, poi eventualmente `KILL`.
- Uscire dal loop dopo shutdown.
- Gestire pidfile mancanti o stale.

### 7. PHP minimo dichiarato ormai vecchio

File: `README.md`

Il README richiede `php-cli (>= 7.4)`. Al 2026-06-25 PHP 7.4 e' fuori supporto ufficiale dal 2022-11-28. La pagina ufficiale PHP indica come branch supportati 8.2, 8.3, 8.4 e 8.5; 8.2 riceve solo security fix fino al 2026-12-31, mentre 8.4 e 8.5 hanno supporto piu' lungo.

Intervento consigliato:

- Puntare a PHP 8.4 come baseline moderna e stabile.
- In alternativa usare PHP 8.3 se serve compatibilita' con distro enterprise piu' conservative.
- Aggiungere `composer.json` con `php: ^8.3 || ^8.4 || ^8.5` o, se si vuole essere piu' selettivi, `php: ^8.4`.
- Aggiungere lint e static analysis in CI.

Fonti PHP consultate:

- https://www.php.net/supported-versions.php
- https://www.php.net/eol.php

### 8. Asset frontend caricati da CDN

File: `public/header.php`, `public/footer.php`

Bootstrap, Popper e Material Icons vengono caricati da CDN (`public/header.php:11-13`, `public/footer.php:1-2`). Per uno strumento locale di amministrazione server questo significa:

- l'app dipende da internet per renderizzare correttamente;
- la prima apertura puo' fallire offline;
- si espone a tracking/contatti esterni non necessari;
- la versione Bootstrap 5.0.2 e' datata.

Intervento consigliato:

- Vendorizzare asset minimi in `public/vendor/` o passare a build frontend con package manager.
- Aggiornare Bootstrap a una versione corrente solo dopo aver separato UI e logica.
- Sostituire Material Icons CDN con icone locali o SVG/componenti.

### 9. Duplicazione del rendering per host, LXC, VM e Docker

File: `public/hosts.php`, `public/lxc.php`, `public/vm.php`, `public/docker.php`

Le quattro viste ripetono quasi lo stesso markup e gli stessi handler. Questo rende facile correggere un bug in un tipo di riga e dimenticarlo negli altri.

Intervento consigliato:

- Creare una funzione/template unico per una "resource row".
- Modellare `host`, `lxc`, `vm`, `docker` come tipi di una stessa entita' con campi comuni.
- Conservare colori e icone per tipo, ma evitare copie del markup.

### 10. Funzionalita' delete ambigua

File: `public/lib/delete_host.php`

La cancellazione usa solo `hostname` come identificatore (`delete_host.php:4`, `delete_host.php:11`, `delete_host.php:21`). Se due host/subsystem hanno lo stesso hostname, la cancellazione puo' rimuovere l'elemento sbagliato o piu' elementi del previsto. Inoltre stampa debug HTML prima del redirect (`delete_host.php:12-14`), cosa che puo' rompere gli header.

Intervento consigliato:

- Introdurre ID stabili univoci.
- Distinguere cancellazione host da cancellazione subsystem.
- Rimuovere output debug.
- Reindicizzare array JSON dopo `unset()` oppure usare ID e filtri espliciti.

## Priorita' bassa ma utile

### 11. Struttura progetto non standard

Mancano:

- `composer.json`;
- test automatici;
- CI;
- formatter/linter;
- file di configurazione applicativa documentato;
- script non interattivi per installazione o aggiornamento;
- packaging Linux (`.deb`, AppImage, Flatpak, systemd user service o desktop entry robusta).

Intervento consigliato:

- Introdurre Composer anche senza framework, almeno per autoload, test e tooling.
- Separare `src/`, `public/`, `var/` o `data/`.
- Spostare `public/db` fuori da `public`, anche se il server integrato espone solo la document root.
- Aggiungere un comando `bin/bashes` o `scripts/start`.

### 12. UX locale da aggiornare senza cambiare natura

L'app e' funzionale, ma molto legata a Bootstrap base e azioni immediate. Miglioramenti coerenti con il progetto:

- ricerca piu' ricca per hostname, IP, user, tipo;
- edit host/subsystem;
- duplicazione host/subsystem;
- tag/gruppi/ambienti;
- note locali per host;
- stato reachability opzionale (`ping`/porta SSH) con caching;
- import/export JSON;
- backup automatico prima di ogni scrittura;
- comando "open Proxmox" configurabile per porta/protocollo;
- preferenze per terminale, file manager, browser.

## Electron: fattibilita' e compatibilita' Windows/macOS

Passare a Electron cambierebbe il packaging e l'integrazione desktop, ma non rende automaticamente tutte le funzionalita' compatibili con Windows e macOS.

### Cosa diventerebbe piu' portabile

- La UI girerebbe in una finestra desktop nativa su Linux, Windows e macOS.
- Si potrebbero distribuire installer/app bundle.
- Le preferenze locali e il datastore potrebbero essere gestiti in una directory app standard.
- Il backend potrebbe diventare Node.js invece di PHP, evitando la dipendenza da `php-cli`.

### Cosa resta specifico per sistema operativo

- Aprire terminali SSH richiede implementazioni diverse:
  - Linux: `gnome-terminal`, `konsole`, `xterm`, ecc.
  - macOS: Terminal.app o iTerm2 via `open`/AppleScript.
  - Windows: Windows Terminal, PowerShell, OpenSSH client.
- Aprire filesystem remoto via `ssh://` non ha comportamento uniforme:
  - Linux puo' usare GVFS/Nautilus/Nemo;
  - macOS ha strumenti diversi e spesso serve SFTP client dedicato;
  - Windows non monta `ssh://` nello stesso modo senza componenti aggiuntivi.
- `ssh-copy-id` non e' sempre disponibile su Windows.
- La gestione delle chiavi SSH cambia tra OpenSSH, agent, keychain e integrazioni di sistema.

### Valutazione pragmatica

Electron e' sensato se l'obiettivo diventa distribuire una vera app desktop multipiattaforma. Pero' il primo rinnovo puo' restare web app locale, correggendo sicurezza, datastore, script e UI. Questo riduce il rischio e mantiene il progetto riconoscibile.

Una strada equilibrata:

1. Rinnovare l'attuale app PHP locale con confini sicuri e PHP 8.4.
2. Separare il dominio applicativo dalla UI: datastore, validazione, comandi desktop.
3. Solo dopo decidere se:
   - restare PHP + browser locale;
   - passare a backend Node/Tauri/Electron;
   - offrire sia web locale che desktop wrapper.

Se si vuole massima portabilita' con peso minore rispetto a Electron, valutare anche Tauri. Richiede pero' riscrivere parte dell'integrazione nativa in Rust/command sidecar e non elimina le differenze SSH/file manager tra sistemi.

## Nuovo requisito: terminale e SSH integrati

Obiettivo aggiornato: l'app dovrebbe diventare il piu' possibile autoconsistente, includendo un emulatore terminale web-based e una gestione SSH interna, senza dipendere da terminale, file manager, browser o client SSH gia' installati nel sistema operativo.

Implicazioni:

- Il frontend puo' usare un terminale web maturo, ad esempio `xterm.js`.
- Il backend non dovrebbe piu' lanciare `gnome-terminal`, `nemo`, `ssh` o `ssh-copy-id` come programmi esterni per il flusso principale.
- Serve un client SSH implementato nel backend dell'app, con gestione di sessioni, resize PTY, input/output streaming, autenticazione con password/chiave, known_hosts e gestione sicura delle credenziali.
- SFTP/SCP va considerato come funzionalita' interna se si vuole sostituire davvero l'apertura del filesystem remoto.
- La portabilita' cross-platform migliora, ma aumenta il peso del core applicativo: Linux, macOS e Windows hanno differenze su storage delle chiavi, agent SSH, keychain e permessi.

Con questo requisito, la shortlist cambia:

- Tauri resta valido se si accetta un backend Rust e si sceglie una libreria SSH Rust o un sidecar controllato.
- Wails diventa una alternativa molto interessante: UI web leggera, backend Go, binari nativi, e lo stack Go ha librerie SSH solide.
- Electron resta scartato per ora per peso e consumo, anche se tecnicamente avrebbe molte librerie Node gia' pronte per SSH e terminali.
- Una web app PHP locale non e' piu' l'architettura ideale se deve includere terminale interattivo e SSH completo cross-platform.

Architettura target consigliata:

1. UI web locale con `xterm.js` per il terminale.
2. Backend nativo con API interne per host, sessioni SSH, SFTP e preferenze.
3. Canale bidirezionale tra terminale e backend tramite IPC/WebSocket/event stream.
4. Datastore locale indipendente dal frontend.
5. Adattatori OS solo per funzioni davvero native, come keychain, notifiche, tray, apertura link e packaging.

## Roadmap consigliata

### Fase 1: hardening senza snaturare

- Validazione input.
- Escaping HTML/JS.
- Command execution sicura.
- Token locale anti-CSRF.
- JSON atomico con backup.
- Rimozione path hardcoded.
- README aggiornato a PHP 8.4/8.3+.

### Fase 2: pulizia architetturale

- `composer.json` e autoload.
- `src/Repository/HostsRepository.php`.
- `src/Service/SshLauncher.php`.
- `src/Http` minimale o micro-framework leggero.
- Template condivisi.
- Test su repository JSON e validatori.

### Fase 3: UX e packaging Linux

- Preferenze desktop.
- Import/export/backup.
- Modifica host/subsystem.
- Desktop entry robusta.
- Installer non interattivo.
- Eventuale systemd user service o AppImage/Flatpak.

### Fase 4: scelta desktop multipiattaforma

- Proof of concept Electron o Tauri.
- Matrice comandi SSH per Linux/macOS/Windows.
- Strategia per filesystem remoto.
- Packaging e auto-update solo quando il core e' gia' stabile.

## Verifiche eseguite

- Letta la struttura completa del repository.
- Analizzati script shell, file PHP, JS, CSS e README.
- Cercati pattern rischiosi: `exec()`, `$_REQUEST`, `file_put_contents()`, CDN, path hardcoded, dipendenze desktop.
- Verificata assenza di manifest moderni (`composer.json`, `package.json`, test config, Dockerfile, Makefile).
- Tentato `php -v` e lint PHP, ma nell'ambiente corrente `php` non e' installato.
