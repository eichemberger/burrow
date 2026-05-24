package tui

import (
	"fmt"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/eichemberger/burrow/internal/apictx"
	"github.com/eichemberger/burrow/internal/awsconfig"
	"github.com/eichemberger/burrow/internal/configstore"
	"github.com/eichemberger/burrow/internal/debuglog"
	"github.com/eichemberger/burrow/internal/services"
	"github.com/eichemberger/burrow/internal/ssmexec"
	"github.com/eichemberger/burrow/internal/targetstore"
	"github.com/eichemberger/burrow/internal/tui/steps"
	"github.com/eichemberger/burrow/internal/ui"
)

type Step int

const (
	StepTargetsRecovery Step = iota
	StepSetup
	StepHome
	StepConnectKnown
	StepManage
	StepTargetAction
	StepTargetEdit
	StepTargetDelete
	StepProfile
	StepRegion
	StepService
	StepResource
	StepEndpoint
	StepManual
	StepBastion
	StepLocalPort
	StepSaveTarget
	StepRun
	StepSessionError
)

type App struct {
	step    Step
	session Session
	opts    Options
	store   *targetstore.Store
	config  *configstore.Config

	fromHome   bool
	fromManage bool
	child      tea.Model
	width      int
	height     int
	pulse      int
	err        error
}

type frameTickMsg struct{}

func NewApp(opts Options) (*App, tea.Cmd) {
	if opts.BurrowDir == "" {
		opts.BurrowDir = targetstore.DefaultDir()
	}

	store, err := targetstore.Load(opts.BurrowDir)
	if err != nil {
		if targetstore.NeedsRecovery(err) {
			app := &App{
				step:    StepTargetsRecovery,
				session: Session{AWSDir: opts.AWSDir},
				opts:    opts,
			}
			app.child = steps.NewTargetsRecoveryModel(opts.BurrowDir, err)
			return app, app.child.Init()
		}
		return &App{err: err, opts: opts}, nil
	}

	app := &App{
		session: Session{AWSDir: opts.AWSDir},
		opts:    opts,
		store:   store,
	}
	return app, app.startAfterTargetsLoaded()
}

func (a *App) startAfterTargetsLoaded() tea.Cmd {
	cfg, cfgErr := configstore.Load(a.opts.BurrowDir)
	if configstore.NeedsSetup(cfgErr) {
		a.step = StepSetup
		a.child = steps.NewSetupModel(a.opts.BurrowDir, cfgErr)
		return a.child.Init()
	}
	if cfgErr != nil {
		a.err = cfgErr
		return nil
	}

	a.config = cfg
	a.step = StepHome
	if a.opts.Manage {
		a.step = StepManage
		a.child = steps.NewManageTargetsModel(a.store)
	} else {
		a.child = steps.NewHomeModel(a.store)
	}
	return a.child.Init()
}

func (a *App) reloadStore() error {
	store, err := targetstore.Load(a.opts.BurrowDir)
	if err != nil {
		return err
	}
	a.store = store
	return nil
}

func (a *App) startWizard() tea.Cmd {
	if a.opts.Profile != "" {
		a.session.Profile = a.opts.Profile
		a.session.UseEnv = false
		if a.opts.Region != "" {
			a.session.Region = a.opts.Region
			return a.loadConfig()
		}
		a.step = StepRegion
		a.child = steps.NewRegionModel(a.session.AWSDir, a.opts.Profile, "")
		return a.child.Init()
	}

	a.step = StepProfile
	child, _ := steps.NewProfileModel(a.session.AWSDir, "")
	a.child = child
	return a.child.Init()
}

func (a *App) Init() tea.Cmd {
	if a.err != nil {
		return frameTick()
	}
	cmds := []tea.Cmd{frameTick()}
	if a.child != nil {
		cmds = append(cmds, a.child.Init())
	}
	if a.session.Region != "" && (a.opts.Profile != "" || a.session.UseEnv) {
		cmds = append(cmds, a.loadConfig())
	}
	return tea.Batch(cmds...)
}

func (a *App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if a.err != nil {
		if cmd, handled := steps.HandleQuitOnly(msg); handled {
			return a, cmd
		}
		return a, nil
	}

	switch msg := msg.(type) {
	case steps.QuitMsg:
		return a, tea.Quit

	case frameTickMsg:
		a.pulse++
		return a, frameTick()

	case steps.TargetsResetCompleteMsg:
		a.store = msg.Store
		return a, a.startAfterTargetsLoaded()

	case steps.SetupCompleteMsg:
		a.config = msg.Config
		if a.opts.Manage {
			return a.transitionTo(steps.NewManageTargetsModel(a.store))
		}
		return a.transitionTo(steps.NewHomeModel(a.store))

	case steps.NewConnectionSelected:
		a.fromHome = !msg.ReturnToManage
		a.fromManage = msg.ReturnToManage
		return a, a.startWizard()

	case steps.ManageConnectionsSelected:
		return a.transitionTo(steps.NewManageTargetsModel(a.store))

	case steps.ConnectKnownSelected:
		return a.transitionTo(steps.NewConnectKnownModel(a.store))

	case steps.TargetManageActionRequested:
		return a.transitionTo(steps.NewTargetActionModel(msg.Alias, msg.Target))

	case steps.TargetEditRequested:
		return a.transitionTo(steps.NewTargetEditModel(a.store, msg.Alias, msg.Target))

	case steps.TargetDeleteRequested:
		return a.transitionTo(steps.NewTargetDeleteModel(a.store, msg.Alias, msg.Target))

	case steps.TargetSavedMsg:
		if err := a.reloadStore(); err != nil {
			a.err = err
			return a, nil
		}
		return a.transitionTo(steps.NewManageTargetsModel(a.store))

	case steps.TargetDeletedMsg:
		if err := a.reloadStore(); err != nil {
			a.err = err
			return a, nil
		}
		return a.transitionTo(steps.NewManageTargetsModel(a.store))

	case steps.SavedTargetSelected:
		return a.transitionTo(steps.NewRunModel(a.session.AWSDir, msg.Target.ToSSMExec(), true))

	case steps.RunFailedMsg:
		return a.transitionTo(steps.NewSessionErrorModel(msg.Failure, msg.FromSaved))

	case steps.GoHomeMsg:
		return a.transitionTo(steps.NewHomeModel(a.store))

	case steps.ConfigLoadedMsg:
		if msg.Err != nil {
			a.err = msg.Err
			a.step = StepRegion
			a.child = steps.NewRegionModel(a.session.AWSDir, a.session.Profile, a.session.Region)
			return a, a.child.Init()
		}
		a.session.AWSConfig = msg.Config
		a.err = nil
		return a.transitionTo(steps.NewServiceModel())

	case steps.AuthModeSelected:
		a.session.UseEnv = msg.UseEnv
		if msg.UseEnv {
			a.session.Profile = ""
		}
		if a.opts.Region != "" {
			a.session.Region = a.opts.Region
			return a, a.loadConfig()
		}
		a.step = StepRegion
		a.child = steps.NewRegionModel(a.session.AWSDir, a.session.Profile, a.opts.Region)
		return a, a.child.Init()

	case steps.ProfileSelected:
		a.session.Profile = msg.Profile
		a.session.UseEnv = false
		if a.opts.Region != "" {
			a.session.Region = a.opts.Region
			return a, a.loadConfig()
		}
		a.step = StepRegion
		a.child = steps.NewRegionModel(a.session.AWSDir, a.session.Profile, a.opts.Region)
		return a, a.child.Init()

	case steps.RegionSelected:
		a.session.Region = msg.Region
		return a, a.loadConfig()

	case steps.ServiceSelected:
		if msg.Manual {
			a.session.Provider = nil
			return a.transitionTo(steps.NewManualModel())
		}
		provider := findProvider(msg.ProviderName)
		if provider == nil {
			a.err = fmt.Errorf("unknown provider %q", msg.ProviderName)
			return a.transitionTo(steps.NewServiceModel())
		}
		a.session.Provider = provider
		return a.transitionTo(steps.NewResourceModel(provider, a.session.AWSConfig))

	case steps.ResourceSelected:
		a.session.Resource = msg.Resource
		if len(msg.Resource.Endpoints) == 1 {
			a.session.Endpoint = msg.Resource.Endpoints[0]
			a.session.Target = msg.Resource.Endpoints[0].Target
			return a.transitionTo(steps.NewBastionModel(a.session.AWSConfig, a.session.Target, a.ec2Selector()))
		}
		return a.transitionTo(steps.NewEndpointModel(msg.Resource))

	case steps.EndpointSelected:
		a.session.Endpoint = msg.Endpoint
		a.session.Target = msg.Endpoint.Target
		return a.transitionTo(steps.NewBastionModel(a.session.AWSConfig, a.session.Target, a.ec2Selector()))

	case steps.ManualTargetEntered:
		a.session.Target = msg.Target
		return a.transitionTo(steps.NewBastionModel(a.session.AWSConfig, a.session.Target, a.ec2Selector()))

	case steps.BastionSelected:
		debuglog.Printf("bastion selected id=%s ip=%s", msg.Bastion.ID, msg.Bastion.PrivateIP)
		a.session.Bastion = msg.Bastion
		return a.transitionTo(steps.NewLocalPortModel(a.session.Target))

	case steps.LocalPortEntered:
		a.session.LocalPort = msg.Port
		return a.transitionTo(steps.NewSaveTargetModel(a.store))

	case steps.TargetSaveEntered:
		if msg.Alias != "" {
			if err := a.store.Set(msg.Alias, sessionToTarget(a.session, msg.Description)); err != nil {
				a.err = err
				return a, nil
			}
			_ = a.reloadStore()
		}
		return a.transitionTo(steps.NewRunModel(a.session.AWSDir, a.sessionSSMOptions(), false))

	case steps.BackMsg:
		return a.goBack()

	case tea.WindowSizeMsg:
		a.width = msg.Width
		a.height = msg.Height
		a.applyChildSize()
	}

	if a.child == nil {
		return a, nil
	}

	child, cmd := a.child.Update(msg)
	a.child = child
	return a, cmd
}

func (a *App) sessionSSMOptions() ssmexec.Options {
	return ssmexec.Options{
		TargetInstanceID: a.session.Bastion.ID,
		Host:             a.session.Target.Host,
		RemotePort:       a.session.Target.Port,
		LocalPort:        a.session.LocalPort,
		Profile:          a.session.Profile,
		Region:           a.session.Region,
		UseEnv:           a.session.UseEnv,
	}
}

func (a *App) View() string {
	breadcrumb := ui.WizardBreadcrumb(a.wizardStep())

	var body string
	switch {
	case a.err != nil:
		body = ui.ErrorStyle.Render("✗ "+a.err.Error()) + "\n\n" + ui.HelpStyle.Render(ui.HelpKeys)
	case a.child == nil:
		body = ui.LoadingLine("●", "Loading AWS configuration...")
	default:
		body = a.child.View()
	}

	return ui.Page(a.width, a.height, a.pulse, breadcrumb, body)
}

func (a *App) transitionTo(child tea.Model) (tea.Model, tea.Cmd) {
	a.child = child
	a.step = a.stepForChild(child)
	a.err = nil
	a.applyChildSize()
	return a, child.Init()
}

func (a *App) applyChildSize() {
	if a.width <= 0 || a.child == nil {
		return
	}
	s, ok := a.child.(steps.Sizable)
	if !ok {
		return
	}
	hasBreadcrumb := a.wizardStep() != ui.WizardNone
	w, h := ui.PageDimensions(a.width, a.height, hasBreadcrumb)
	s.SetSize(w, h)
}

func (a *App) wizardStep() ui.WizardStep {
	switch a.step {
	case StepProfile:
		return ui.WizardAuth
	case StepRegion:
		return ui.WizardRegion
	case StepService:
		return ui.WizardService
	case StepResource, StepManual:
		return ui.WizardResource
	case StepEndpoint:
		return ui.WizardEndpoint
	case StepBastion:
		return ui.WizardBastion
	case StepLocalPort:
		return ui.WizardLocalPort
	case StepSaveTarget:
		return ui.WizardSave
	case StepRun:
		return ui.WizardConnect
	}
	return ui.WizardNone
}

func frameTick() tea.Cmd {
	return tea.Tick(140*time.Millisecond, func(time.Time) tea.Msg {
		return frameTickMsg{}
	})
}

func (a *App) stepForChild(child tea.Model) Step {
	switch child.(type) {
	case *steps.TargetsRecoveryModel:
		return StepTargetsRecovery
	case *steps.SetupModel:
		return StepSetup
	case *steps.HomeModel:
		return StepHome
	case *steps.ConnectKnownModel:
		return StepConnectKnown
	case *steps.ManageTargetsModel:
		return StepManage
	case *steps.TargetActionModel:
		return StepTargetAction
	case *steps.TargetEditModel:
		return StepTargetEdit
	case *steps.TargetDeleteModel:
		return StepTargetDelete
	case *steps.ProfileModel:
		return StepProfile
	case *steps.RegionModel:
		return StepRegion
	case *steps.ServiceModel:
		return StepService
	case *steps.ResourceModel:
		return StepResource
	case *steps.EndpointModel:
		return StepEndpoint
	case *steps.ManualModel:
		return StepManual
	case *steps.BastionModel:
		return StepBastion
	case *steps.LocalPortModel:
		return StepLocalPort
	case *steps.SaveTargetModel:
		return StepSaveTarget
	case *steps.SessionErrorModel:
		return StepSessionError
	case *steps.RunModel:
		return StepRun
	default:
		return a.step
	}
}

func (a *App) goBack() (tea.Model, tea.Cmd) {
	switch a.step {
	case StepHome:
		return a, tea.Quit
	case StepConnectKnown:
		return a.transitionTo(steps.NewHomeModel(a.store))
	case StepManage:
		return a.transitionTo(steps.NewHomeModel(a.store))
	case StepTargetAction:
		return a.transitionTo(steps.NewManageTargetsModel(a.store))
	case StepTargetEdit:
		return a.transitionTo(steps.NewManageTargetsModel(a.store))
	case StepTargetDelete:
		return a.transitionTo(steps.NewManageTargetsModel(a.store))
	case StepProfile:
		if a.fromManage {
			a.fromManage = false
			return a.transitionTo(steps.NewManageTargetsModel(a.store))
		}
		if a.fromHome {
			a.fromHome = false
			return a.transitionTo(steps.NewHomeModel(a.store))
		}
		return a, tea.Quit
	case StepRegion:
		if a.opts.Profile != "" {
			if a.fromManage {
				a.fromManage = false
				return a.transitionTo(steps.NewManageTargetsModel(a.store))
			}
			if a.fromHome {
				a.fromHome = false
				return a.transitionTo(steps.NewHomeModel(a.store))
			}
			return a, tea.Quit
		}
		child, _ := steps.NewProfileModel(a.session.AWSDir, a.opts.Profile)
		a.child = child
		a.step = StepProfile
		return a, child.Init()
	case StepService:
		if a.opts.Region != "" {
			if a.opts.Profile != "" {
				if a.fromManage {
					a.fromManage = false
					return a.transitionTo(steps.NewManageTargetsModel(a.store))
				}
				if a.fromHome {
					a.fromHome = false
					return a.transitionTo(steps.NewHomeModel(a.store))
				}
				return a, tea.Quit
			}
			child, _ := steps.NewProfileModel(a.session.AWSDir, a.opts.Profile)
			a.child = child
			a.step = StepProfile
			return a, child.Init()
		}
		a.child = steps.NewRegionModel(a.session.AWSDir, a.session.Profile, a.opts.Region)
		a.step = StepRegion
		return a, a.child.Init()
	case StepResource:
		return a.transitionTo(steps.NewServiceModel())
	case StepEndpoint:
		return a.transitionTo(steps.NewResourceModel(a.session.Provider, a.session.AWSConfig))
	case StepManual:
		return a.transitionTo(steps.NewServiceModel())
	case StepBastion:
		if len(a.session.Resource.Endpoints) > 1 {
			return a.transitionTo(steps.NewEndpointModel(a.session.Resource))
		}
		if a.session.Provider != nil {
			return a.transitionTo(steps.NewResourceModel(a.session.Provider, a.session.AWSConfig))
		}
		return a.transitionTo(steps.NewManualModel())
	case StepLocalPort:
		return a.transitionTo(steps.NewBastionModel(a.session.AWSConfig, a.session.Target, a.ec2Selector()))
	case StepSaveTarget:
		return a.transitionTo(steps.NewLocalPortModel(a.session.Target))
	case StepRun:
		return a.transitionTo(steps.NewSaveTargetModel(a.store))
	default:
		return a, nil
	}
}

func (a *App) loadConfig() tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := apictx.AuthBackground()
		defer cancel()

		cfg, err := awsconfig.Load(ctx, awsconfig.Options{
			AWSDir:  a.session.AWSDir,
			Profile: a.session.Profile,
			Region:  a.session.Region,
			UseEnv:  a.session.UseEnv,
		})
		return steps.ConfigLoadedMsg{Config: cfg, Err: apictx.WrapDeadline(err, "load AWS configuration")}
	}
}

func findProvider(name string) services.Provider {
	for _, provider := range services.All() {
		if provider.Name() == name {
			return provider
		}
	}
	return nil
}

func (a *App) ec2Selector() *configstore.EC2Selector {
	if a.config == nil {
		return nil
	}
	ec2, err := a.config.EC2()
	if err != nil {
		return nil
	}
	return ec2
}

func Run(opts Options) error {
	app, _ := NewApp(opts)
	p := tea.NewProgram(app, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		return err
	}
	return nil
}
